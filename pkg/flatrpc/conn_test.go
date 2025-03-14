// Copyright 2024 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package flatrpc

import (
	"context"
	"fmt"
	"net"
	"os"
	"reflect"
	"runtime/debug"
	"sync"
	"syscall"
	"testing"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/stretchr/testify/assert"
)

func TestConn(t *testing.T) {
	connectHello := &ConnectHello{
		Cookie: 1,
	}
	connectReq := &ConnectRequest{
		Cookie:      73856093,
		Id:          1,
		Arch:        "arch",
		GitRevision: "rev1",
		SyzRevision: "rev2",
	}
	connectReply := &ConnectReply{
		LeakFrames: []string{"foo", "bar"},
		RaceFrames: []string{"bar", "baz"},
		Features:   FeatureCoverage | FeatureLeak,
		Files:      []string{"file1"},
	}
	executorMsg := &ExecutorMessage{
		Msg: &ExecutorMessages{
			Type: ExecutorMessagesRawExecuting,
			Value: &ExecutingMessage{
				Id:     1,
				ProcId: 2,
				Try:    3,
			},
		},
	}

	serv, err := Listen(":0")
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error)
	go func() {
		done <- serv.Serve(context.Background(),
			func(_ context.Context, c *Conn) error {
				if err := Send(c, connectHello); err != nil {
					return err
				}
				connectReqGot, err := Recv[*ConnectRequestRaw](c)
				if err != nil {
					return err
				}
				if !reflect.DeepEqual(connectReq, connectReqGot) {
					return fmt.Errorf("connectReq != connectReqGot")
				}

				if err := Send(c, connectReply); err != nil {
					return err
				}

				for i := 0; i < 10; i++ {
					got, err := Recv[*ExecutorMessageRaw](c)
					if err != nil {
						return nil
					}
					if !reflect.DeepEqual(executorMsg, got) {
						return fmt.Errorf("executorMsg !=got")
					}
				}
				return nil
			})
	}()
	c := dial(t, serv.Addr.String())
	defer c.Close()

	connectHelloGot, err := Recv[*ConnectHelloRaw](c)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, connectHello, connectHelloGot)

	if err := Send(c, connectReq); err != nil {
		t.Fatal(err)
	}

	connectReplyGot, err := Recv[*ConnectReplyRaw](c)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, connectReply, connectReplyGot)

	for i := 0; i < 10; i++ {
		if err := Send(c, executorMsg); err != nil {
			t.Fatal(err)
		}
	}

	serv.Close()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func BenchmarkConn(b *testing.B) {
	connectHello := &ConnectHello{
		Cookie: 1,
	}
	connectReq := &ConnectRequest{
		Cookie:      73856093,
		Id:          1,
		Arch:        "arch",
		GitRevision: "rev1",
		SyzRevision: "rev2",
	}
	connectReply := &ConnectReply{
		LeakFrames: []string{"foo", "bar"},
		RaceFrames: []string{"bar", "baz"},
		Features:   FeatureCoverage | FeatureLeak,
		Files:      []string{"file1"},
	}

	serv, err := Listen(":0")
	if err != nil {
		b.Fatal(err)
	}
	done := make(chan error)

	go func() {
		done <- serv.Serve(context.Background(),
			func(_ context.Context, c *Conn) error {
				for i := 0; i < b.N; i++ {
					if err := Send(c, connectHello); err != nil {
						return err
					}

					_, err = Recv[*ConnectRequestRaw](c)
					if err != nil {
						return err
					}
					if err := Send(c, connectReply); err != nil {
						return err
					}
				}
				return nil
			})
	}()

	c := dial(b, serv.Addr.String())
	defer c.Close()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Recv[*ConnectHelloRaw](c)
		if err != nil {
			b.Fatal(err)
		}
		if err := Send(c, connectReq); err != nil {
			b.Fatal(err)
		}
		_, err = Recv[*ConnectReplyRaw](c)
		if err != nil {
			b.Fatal(err)
		}
	}

	serv.Close()
	if err := <-done; err != nil {
		b.Fatal(err)
	}
}

func dial(t testing.TB, addr string) *Conn {
	conn, err := net.DialTimeout("tcp", addr, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	return NewConn(conn)
}

var memoryLimitOnce sync.Once

func FuzzRecv(f *testing.F) {
	msg := &ExecutorMessage{
		Msg: &ExecutorMessages{
			Type: ExecutorMessagesRawExecResult,
			Value: &ExecResult{
				Id:     1,
				Output: []byte("aaa"),
				Error:  "bbb",
				Info: &ProgInfo{
					ExtraRaw: []*CallInfo{
						{
							Signal: []uint64{1, 2},
						},
					},
				},
			},
		},
	}
	builder := flatbuffers.NewBuilder(0)
	builder.FinishSizePrefixed(msg.Pack(builder))
	f.Add(builder.FinishedBytes())
	f.Fuzz(func(t *testing.T, data []byte) {
		memoryLimitOnce.Do(func() {
			debug.SetMemoryLimit(64 << 20)
		})
		if len(data) > 1<<10 {
			t.Skip()
		}
		fds, err := syscall.Socketpair(syscall.AF_LOCAL, syscall.SOCK_STREAM, 0)
		if err != nil {
			t.Fatal(err)
		}
		w := os.NewFile(uintptr(fds[0]), "")
		r := os.NewFile(uintptr(fds[1]), "")
		defer w.Close()
		defer r.Close()
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
		w.Close()
		n, err := net.FileConn(r)
		if err != nil {
			t.Fatal(err)
		}
		c := NewConn(n)
		for {
			_, err := Recv[*ExecutorMessageRaw](c)
			if err != nil {
				break
			}
		}
	})
}
