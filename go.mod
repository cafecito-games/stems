module github.com/cafecito-games/stems

go 1.26.0

retract v0.1.0 // Module zip includes Git LFS files under testdata/, causing checksum mismatches across hosts with/without git-lfs.

require github.com/stretchr/testify v1.12.1

require go.yaml.in/yaml/v3 v3.0.5 // indirect
