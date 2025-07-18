// Copyright 2016 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

// TODO: strip " (discriminator N)", "constprop", "isra" from function names.

package symbolizer

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"github.com/google/syzkaller/pkg/osutil"
	"github.com/google/syzkaller/sys/targets"
)

type addr2Line struct {
	target   *targets.Target
	subprocs map[string]*subprocess
	interner Interner
}

type subprocess struct {
	cmd     *exec.Cmd
	stdin   io.Closer
	stdout  io.Closer
	input   *bufio.Writer
	scanner *bufio.Scanner
}

func (s *addr2Line) Symbolize(bin string, pcs ...uint64) ([]Frame, error) {
	sub, err := s.getSubprocess(bin)
	if err != nil {
		return nil, err
	}
	return symbolize(&s.interner, sub.input, sub.scanner, pcs)
}

func (s *addr2Line) Close() {
	for _, sub := range s.subprocs {
		sub.stdin.Close()
		sub.stdout.Close()
		sub.cmd.Process.Kill()
		sub.cmd.Wait()
	}
}

func (s *addr2Line) getSubprocess(bin string) (*subprocess, error) {
	if sub := s.subprocs[bin]; sub != nil {
		return sub, nil
	}
	addr2line, err := s.target.Addr2Line()
	if err != nil {
		return nil, err
	}
	cmd := osutil.Command(addr2line, "-afi", "-e", bin)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		return nil, err
	}
	sub := &subprocess{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		input:   bufio.NewWriter(stdin),
		scanner: bufio.NewScanner(stdout),
	}
	if s.subprocs == nil {
		s.subprocs = make(map[string]*subprocess)
	}
	s.subprocs[bin] = sub
	return sub, nil
}

func symbolize(interner *Interner, input *bufio.Writer, scanner *bufio.Scanner, pcs []uint64) ([]Frame, error) {
	var frames []Frame
	done := make(chan error, 1)
	go func() {
		var err error
		defer func() {
			done <- err
		}()
		if !scanner.Scan() {
			if err = scanner.Err(); err == nil {
				err = io.EOF
			}
			return
		}
		for range pcs {
			var frames1 []Frame
			frames1, err = parse(interner, scanner)
			if err != nil {
				return
			}
			frames = append(frames, frames1...)
		}
		for i := 0; i < 2; i++ {
			scanner.Scan()
		}
	}()

	for _, pc := range pcs {
		if _, err := fmt.Fprintf(input, "0x%x\n", pc); err != nil {
			return nil, err
		}
	}
	// Write an invalid PC so that parse doesn't block reading input.
	// We read out result for this PC at the end of the function.
	if _, err := fmt.Fprintf(input, "0xffffffffffffffff\n"); err != nil {
		return nil, err
	}
	input.Flush()

	if err := <-done; err != nil {
		return nil, err
	}
	return frames, nil
}

func parse(interner *Interner, s *bufio.Scanner) ([]Frame, error) {
	pc, err := strconv.ParseUint(s.Text(), 0, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pc '%v' in addr2line output: %w", s.Text(), err)
	}
	var frames []Frame
	for s.Scan() {
		ln := s.Text()
		if len(ln) > 3 && ln[0] == '0' && ln[1] == 'x' {
			break
		}
		fn := ln
		if !s.Scan() {
			err := s.Err()
			if err == nil {
				err = io.EOF
			}
			return nil, fmt.Errorf("failed to read file:line from addr2line: %w", err)
		}
		ln = s.Text()
		colon := strings.LastIndexByte(ln, ':')
		if colon == -1 {
			return nil, fmt.Errorf("failed to parse file:line in addr2line output: %v", ln)
		}
		lineEnd := colon + 1
		for lineEnd < len(ln) && ln[lineEnd] >= '0' && ln[lineEnd] <= '9' {
			lineEnd++
		}
		file := ln[:colon]
		line, err := strconv.Atoi(ln[colon+1 : lineEnd])
		if err != nil || fn == "" || fn == "??" || file == "" || file == "??" || line < 0 {
			continue
		}
		if line == 0 {
			line = -1
		}
		frames = append(frames, Frame{
			PC:     pc,
			Func:   interner.Do(fn),
			File:   interner.Do(file),
			Line:   line,
			Inline: true,
		})
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	if len(frames) != 0 {
		frames[len(frames)-1].Inline = false
	}
	return frames, nil
}
