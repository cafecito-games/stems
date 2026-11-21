// This file exists to keep testdata/ out of the published module zip, and for
// no other reason. Deleting it republishes 31MB of vocabulary fixtures to every
// consumer and reintroduces the bug described below.
//
// The vocabulary fixtures here are tracked with Git LFS (see ../.gitattributes).
// Go builds a module's zip from the working tree, so what lands in that zip --
// and therefore the module's go.sum hash -- depended on whether the fetching
// host had git-lfs installed:
//
//	direct fetch, no git-lfs    131-byte pointer files    h1:jMwhNpm...
//	direct fetch, with git-lfs  real 37MB fixtures        h1:/wrGyyF5...
//
// The proxy has no git-lfs, so proxy consumers got the first hash and that is
// what sum.golang.org recorded. Any consumer whose GOPRIVATE or GONOPROXY
// matches this module fetches direct instead, and on a developer workstation
// (git-lfs installed) computed the second. The two cannot both satisfy one
// go.sum, so a repository would pass CI in one job and fail in another with
// "checksum mismatch / SECURITY ERROR" for the same commit.
//
// Go excludes any directory containing a go.mod file from its parent module's
// zip. Declaring testdata/ a module of its own therefore removes the LFS files
// from the zip entirely, which makes the hash identical on every host:
//
//	before  208 files, 0.4MB or 31.1MB, two different hashes
//	after   168 files, 0.4MB,           one hash everywhere
//
// Nothing imports this module and it is never published. The fixtures stay
// exactly where they were on disk, and internal/testutil/vocab.go still reaches
// them by filesystem path, which does not care about module boundaries.
module github.com/cafecito-games/stems/testdata

go 1.26.0
