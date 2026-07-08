package main

import (
	"flag"
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"os"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func main() {
	flag.Parse()
	
	patterns := flag.Args()
	fmt.Printf("DEBUG: Patterns from args: %v\n", patterns)
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes |
			packages.NeedSyntax | packages.NeedTypesInfo,
		Tests: false,
	}
	
	fmt.Println("Loading packages...")
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load: %v\n", err)
		os.Exit(1)
	}
	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(1)
	}

	fmt.Printf("Loaded %d initial packages:\n", len(pkgs))

	fmt.Println("Building SSA...")
	prog, _ := ssautil.AllPackages(pkgs, ssa.InstantiateGenerics)
	prog.Build()

	fmt.Println("Analyzing functions...")
	count := 0
	for fn := range ssautil.AllFunctions(prog) {
		analyzeFunc(fn)
		count++
	}
	fmt.Printf("Analyzed %d functions.\n", count)
}

func analyzeFunc(fn *ssa.Function) {
	if fn == nil || fn.Blocks == nil {
		return
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if ifInstr, ok := instr.(*ssa.If); ok {
				checkIfCond(fn, ifInstr.Cond)
			}
		}
	}
}

func checkIfCond(fn *ssa.Function, cond ssa.Value) {
	binOp, ok := cond.(*ssa.BinOp)
	if !ok {
		return
	}
	
	var lenVal ssa.Value
	var constVal int64
	var hasConst bool
	var op token.Token = binOp.Op

	// Check for nil comparison first (address-of cannot be nil)
	var isNilCompare bool
	var compareVal ssa.Value

	if constOp, ok := binOp.Y.(*ssa.Const); ok && constOp.IsNil() {
		compareVal = binOp.X
		isNilCompare = true
	} else if constOp, ok := binOp.X.(*ssa.Const); ok && constOp.IsNil() {
		compareVal = binOp.Y
		isNilCompare = true
	}

	if isNilCompare {
		switch v := compareVal.(type) {
		case *ssa.FieldAddr, *ssa.IndexAddr:
			pos := fn.Prog.Fset.Position(binOp.Pos())
			always := ""
			if op == token.EQL {
				always = "FALSE"
			} else if op == token.NEQ {
				always = "TRUE"
			}
			if always != "" {
				fmt.Printf("%s: warning: condition '%s' is always %s because address-of operator target cannot be nil\n",
					pos, formatBinOp(binOp), always)
			}
			return
		case *ssa.Call:
			if isPkgFunc(v.Call.Value, "errors", "New") || isPkgFunc(v.Call.Value, "fmt", "Errorf") {
				pos := fn.Prog.Fset.Position(binOp.Pos())
				always := ""
				if op == token.EQL {
					always = "FALSE"
				} else if op == token.NEQ {
					always = "TRUE"
				}
				if always != "" {
					fmt.Printf("%s: warning: condition '%s' is always %s because errors.New/fmt.Errorf returns a non-nil error\n",
						pos, formatBinOp(binOp), always)
				}
				return
			}
		}
	}

	// Check for empty string comparison: str == "" or str != ""
	var isStrEmptyCompare bool
	var strVal ssa.Value

	if constOp, ok := binOp.Y.(*ssa.Const); ok && constOp.Value != nil && constOp.Value.Kind() == constant.String && constant.StringVal(constOp.Value) == "" {
		strVal = binOp.X
		isStrEmptyCompare = true
	} else if constOp, ok := binOp.X.(*ssa.Const); ok && constOp.Value != nil && constOp.Value.Kind() == constant.String && constant.StringVal(constOp.Value) == "" {
		strVal = binOp.Y
		isStrEmptyCompare = true
	}

	if isStrEmptyCompare {
		if originFnName, isGuaranteedNonEmpty := isSequenceGuaranteedNonEmpty(strVal); isGuaranteedNonEmpty {
			pos := fn.Prog.Fset.Position(binOp.Pos())
			always := ""
			if op == token.EQL {
				always = "FALSE"
			} else if op == token.NEQ {
				always = "TRUE"
			}
			if always != "" {
				fmt.Printf("%s: warning: condition '%s' is always %s because string returned by %s is guaranteed to be non-empty\n",
					pos, formatBinOp(binOp), always, originFnName)
			}
			return
		}
	}

	// Existing slice/string length checks
	if constOp, ok := binOp.Y.(*ssa.Const); ok && constOp.Value != nil {
		if val, err := constToInt(constOp); err == nil {
			constVal = val
			lenVal = binOp.X
			hasConst = true
		}
	} else if constOp, ok := binOp.X.(*ssa.Const); ok && constOp.Value != nil {
		if val, err := constToInt(constOp); err == nil {
			constVal = val
			lenVal = binOp.Y
			hasConst = true
			switch op {
			case token.GTR:
				op = token.LSS
			case token.LSS:
				op = token.GTR
			case token.GEQ:
				op = token.LEQ
			case token.LEQ:
				op = token.GEQ
			}
		}
	}

	if !hasConst {
		return
	}

	// Check if lenVal is a call to built-in len
	call, ok := lenVal.(*ssa.Call)
	if !ok {
		return
	}
	builtin, ok := call.Call.Value.(*ssa.Builtin)
	if !ok || builtin.Name() != "len" {
		return
	}

	if len(call.Call.Args) != 1 {
		return
	}
	seqVal := call.Call.Args[0]

	// Verify it's a slice or string
	underlying := seqVal.Type().Underlying()
	_, isSlice := underlying.(*types.Slice)
	basic, isBasic := underlying.(*types.Basic)
	isString := isBasic && basic.Kind() == types.String
	if !isSlice && !isString {
		return
	}

	// Check if sequence is guaranteed non-empty
	if originFnName, isGuaranteedNonEmpty := isSequenceGuaranteedNonEmpty(seqVal); isGuaranteedNonEmpty {
		pos := fn.Prog.Fset.Position(binOp.Pos())
		
		always := ""
		switch op {
		case token.EQL: // len == 0
			if constVal == 0 {
				always = "FALSE"
			}
		case token.NEQ: // len != 0
			if constVal == 0 {
				always = "TRUE"
			}
		case token.GTR: // len > 0
			if constVal == 0 {
				always = "TRUE"
			}
		case token.GEQ: // len >= 1
			if constVal == 1 {
				always = "TRUE"
			}
		case token.LSS: // len < 1
			if constVal == 1 {
				always = "FALSE"
			}
		}

		if always != "" {
			fmt.Printf("%s: warning: condition '%s' is always %s because sequence returned by %s is guaranteed to be non-empty\n",
				pos, formatBinOp(binOp), always, originFnName)
		}
	}
}

func formatBinOp(binOp *ssa.BinOp) string {
	return fmt.Sprintf("%s %s %s", binOp.X.Name(), binOp.Op, binOp.Y.Name())
}

func constToInt(c *ssa.Const) (int64, error) {
	if c.IsNil() {
		return 0, fmt.Errorf("nil")
	}
	if c.Value.Kind() != constant.Int {
		return 0, fmt.Errorf("not an integer constant")
	}
	return c.Int64(), nil
}

func isSequenceGuaranteedNonEmpty(v ssa.Value) (string, bool) {
	var call *ssa.Call
	var returnIndex int

	if extract, ok := v.(*ssa.Extract); ok {
		if c, ok := extract.Tuple.(*ssa.Call); ok {
			call = c
			returnIndex = extract.Index
		}
	} else if c, ok := v.(*ssa.Call); ok {
		call = c
		returnIndex = 0
	}

	if call == nil {
		return "", false
	}

	// strings and bytes split functions
	if isPkgFunc(call.Call.Value, "strings", "Split") {
		return "strings.Split", true
	}
	if isPkgFunc(call.Call.Value, "strings", "SplitAfter") {
		return "strings.SplitAfter", true
	}
	if isPkgFunc(call.Call.Value, "strings", "SplitN") {
		if len(call.Call.Args) == 3 {
			if constN, ok := call.Call.Args[2].(*ssa.Const); ok {
				if val, err := constToInt(constN); err == nil && val > 0 {
					return "strings.SplitN", true
				}
			}
		}
	}
	if isPkgFunc(call.Call.Value, "strings", "SplitAfterN") {
		if len(call.Call.Args) == 3 {
			if constN, ok := call.Call.Args[2].(*ssa.Const); ok {
				if val, err := constToInt(constN); err == nil && val > 0 {
					return "strings.SplitAfterN", true
				}
			}
		}
	}

	if isPkgFunc(call.Call.Value, "bytes", "Split") {
		return "bytes.Split", true
	}
	if isPkgFunc(call.Call.Value, "bytes", "SplitAfter") {
		return "bytes.SplitAfter", true
	}
	if isPkgFunc(call.Call.Value, "bytes", "SplitN") {
		if len(call.Call.Args) == 3 {
			if constN, ok := call.Call.Args[2].(*ssa.Const); ok {
				if val, err := constToInt(constN); err == nil && val > 0 {
					return "bytes.SplitN", true
				}
			}
		}
	}
	if isPkgFunc(call.Call.Value, "bytes", "SplitAfterN") {
		if len(call.Call.Args) == 3 {
			if constN, ok := call.Call.Args[2].(*ssa.Const); ok {
				if val, err := constToInt(constN); err == nil && val > 0 {
					return "bytes.SplitAfterN", true
				}
			}
		}
	}

	// strconv format functions (always return non-empty strings)
	if isPkgFunc(call.Call.Value, "strconv", "Itoa") {
		return "strconv.Itoa", true
	}
	if isPkgFunc(call.Call.Value, "strconv", "FormatInt") {
		return "strconv.FormatInt", true
	}
	if isPkgFunc(call.Call.Value, "strconv", "FormatUint") {
		return "strconv.FormatUint", true
	}

	fnVal := call.Call.Value
	calledFn, ok := fnVal.(*ssa.Function)
	if !ok {
		return "", false
	}

	if checkFunctionReturnsNonEmpty(calledFn, returnIndex) {
		return calledFn.String(), true
	}
	return "", false
}

func isPkgFunc(v ssa.Value, pkgPath, name string) bool {
	fn, ok := v.(*ssa.Function)
	if !ok {
		return false
	}
	if fn.Pkg != nil && fn.Pkg.Pkg.Path() == pkgPath && fn.Name() == name {
		return true
	}
	if fn.String() == pkgPath+"."+name {
		return true
	}
	return false
}

func checkFunctionReturnsNonEmpty(fn *ssa.Function, returnIndex int) bool {
	if fn.Blocks == nil {
		return false
	}

	var returnInstrs []*ssa.Return
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if ret, ok := instr.(*ssa.Return); ok {
				returnInstrs = append(returnInstrs, ret)
			}
		}
	}

	if len(returnInstrs) == 0 {
		return false
	}

	for _, ret := range returnInstrs {
		if len(ret.Results) <= returnIndex {
			return false
		}
		retVal := ret.Results[returnIndex]
		if !isReturnValueGuaranteedNonEmpty(retVal) {
			return false
		}
	}

	return true
}

func isReturnValueGuaranteedNonEmpty(v ssa.Value) bool {
	if makeSlice, ok := v.(*ssa.MakeSlice); ok {
		if constLen, ok := makeSlice.Len.(*ssa.Const); ok {
			if val, err := constToInt(constLen); err == nil && val > 0 {
				return true
			}
		}
		return false
	}

	if call, ok := v.(*ssa.Call); ok {
		if builtin, ok := call.Call.Value.(*ssa.Builtin); ok && builtin.Name() == "append" {
			if len(call.Call.Args) >= 2 {
				return true
			}
		}
		
		if _, non_empty := isSequenceGuaranteedNonEmpty(v); non_empty {
			return true
		}
	}

	return false
}
