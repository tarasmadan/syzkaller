package main

import (
	"flag"
	"fmt"
	"go/types"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func main() {
	flag.Parse()
	
	patterns := flag.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes |
			packages.NeedSyntax | packages.NeedTypesInfo,
		Tests: false, // production dead code
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

	fmt.Println("Building SSA...")
	prog, _ := ssautil.AllPackages(pkgs, ssa.InstantiateGenerics)
	prog.Build()

	// Collect all struct fields
	structFields := make(map[string]bool) // Key: "pkg.StructName.FieldName"
	fieldPos := make(map[string]string)     // Key -> position string
	
	fmt.Println("Collecting struct fields...")
	for _, pkg := range prog.AllPackages() {
		if pkg == nil || pkg.Pkg == nil {
			continue
		}
		// Only analyze syzkaller packages
		if !strings.HasPrefix(pkg.Pkg.Path(), "github.com/google/syzkaller") {
			continue
		}
		for _, member := range pkg.Members {
			if typeName, ok := member.(*ssa.Type); ok {
				// We look at the named type itself
				named, ok := typeName.Type().(*types.Named)
				if !ok {
					continue
				}
				structType, ok := named.Underlying().(*types.Struct)
				if !ok {
					continue
				}
				
				for i := 0; i < structType.NumFields(); i++ {
					field := structType.Field(i)
					tag := structType.Tag(i)
					
					// Skip fields with common serialization tags
					if hasSerializationTag(tag) {
						continue
					}
					
					key := fmt.Sprintf("%s.%s.%s", pkg.Pkg.Path(), typeName.Name(), field.Name())
					structFields[key] = false // unused initially
					
					pos := prog.Fset.Position(field.Pos())
					fieldPos[key] = pos.String()
				}
			}
		}
	}
	
	fmt.Printf("Collected %d candidate fields (ignoring serialized fields).\n", len(structFields))

	fmt.Println("Tracing field accesses...")
	for fn := range ssautil.AllFunctions(prog) {
		if fn == nil || fn.Blocks == nil {
			continue
		}
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				switch inst := instr.(type) {
				case *ssa.Field:
					markFieldUsed(inst.X.Type(), inst.Field, structFields)
				case *ssa.FieldAddr:
					markFieldUsed(inst.X.Type(), inst.Field, structFields)
				}
			}
		}
	}

	fmt.Println("\nUnused struct fields in production:")
	unusedCount := 0
	for key, used := range structFields {
		if !used {
			fmt.Printf("  %s at %s\n", key, fieldPos[key])
			unusedCount++
		}
	}
	fmt.Printf("\nFound %d unused struct fields.\n", unusedCount)
}

func hasSerializationTag(tag string) bool {
	tags := []string{"json:", "yaml:", "xml:", "protobuf:", "bson:", "bin:"}
	for _, t := range tags {
		if strings.Contains(tag, t) {
			return true
		}
	}
	return false
}

func markFieldUsed(typ types.Type, fieldIdx int, structFields map[string]bool) {
	derefTyp := deref(typ)
	named, ok := derefTyp.(*types.Named)
	if !ok {
		return
	}
	structType, ok := named.Underlying().(*types.Struct)
	if !ok {
		return
	}
	if fieldIdx < 0 || fieldIdx >= structType.NumFields() {
		return
	}
	fieldName := structType.Field(fieldIdx).Name()
	pkgPath := named.Obj().Pkg().Path()
	structName := named.Obj().Name()
	
	key := fmt.Sprintf("%s.%s.%s", pkgPath, structName, fieldName)
	if _, exists := structFields[key]; exists {
		structFields[key] = true
	}
}

func deref(typ types.Type) types.Type {
	if ptr, ok := typ.(*types.Pointer); ok {
		return ptr.Elem()
	}
	return typ
}
