// Copyright 2024 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package triage

import (
	"time"

	"github.com/google/syzkaller/pkg/debugtracer"
	"github.com/google/syzkaller/pkg/vcs"
	"github.com/google/syzkaller/syz-cluster/pkg/api"
)

// TODO: Some further improvements:
//   1. Consider the blob hashes incorporated into the git diff. These may restrict the set of base commits.
//   2. Add support for experimental sessions: these may be way behind the current HEAD.

type TreeOps interface {
	HeadCommit(tree *api.Tree) (*vcs.Commit, error)
	ApplySeries(commit string, patches [][]byte) error
}

type CommitSelector struct {
	ops    TreeOps
	tracer debugtracer.DebugTracer
}

func NewCommitSelector(ops TreeOps, tracer debugtracer.DebugTracer) *CommitSelector {
	return &CommitSelector{ops: ops, tracer: tracer}
}

type SelectResult struct {
	Commit string
	Reason string // Set if Commit is empty.
}

const (
	reasonSeriesTooOld = "series lags behind the current HEAD too much"
	reasonNotApplies   = "series does not apply"
)

// Select returns the best matching commit hash.
func (cs *CommitSelector) Select(series *api.Series, tree *api.Tree, lastBuild *api.Build) (SelectResult, error) {
	head, err := cs.ops.HeadCommit(tree)
	if err != nil || head == nil {
		return SelectResult{}, err
	}
	cs.tracer.Log("current HEAD: %q (%v)", head.Hash, head.Date)
	// If the series is already too old, it may be incompatible even if it applies cleanly.
	const seriesLagsBehind = time.Hour * 24 * 7
	if diff := head.CommitDate.Sub(series.PublishedAt); series.PublishedAt.Before(head.CommitDate) &&
		diff > seriesLagsBehind {
		cs.tracer.Log("the series is too old: %v before the HEAD", diff)
		return SelectResult{Reason: reasonSeriesTooOld}, nil
	}

	// Algorithm:
	// 1. If the last successful build is sufficiently new, prefer it over the last master.
	// We should it be renewing it regularly, so the commit should be quite up to date.
	// 2. If the last build is too old / the series does not apply, give a chance to the
	// current HEAD.

	var hashes []string
	if lastBuild != nil {
		// Check if the commit is still good enough.
		if diff := head.CommitDate.Sub(lastBuild.CommitDate); diff > seriesLagsBehind {
			cs.tracer.Log("the last successful build is already too old: %v, skipping", diff)
		} else {
			hashes = append(hashes, lastBuild.CommitHash)
		}
	}
	for _, hash := range append(hashes, head.Hash) {
		cs.tracer.Log("considering %q", hash)
		err := cs.ops.ApplySeries(hash, series.PatchBodies())
		if err == nil {
			cs.tracer.Log("series can be applied to %q", hash)
			return SelectResult{Commit: hash}, nil
		} else {
			cs.tracer.Log("failed to apply to %q: %v", hash, err)
		}
	}
	return SelectResult{Reason: reasonNotApplies}, nil
}
