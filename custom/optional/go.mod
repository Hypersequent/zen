module github.com/hypersequent/zen/custom/optional

go 1.27

replace github.com/hypersequent/zen => ../..

require (
	4d63.com/optional v0.2.0
	github.com/hypersequent/zen v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.12.1
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect
