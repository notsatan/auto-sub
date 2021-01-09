# Sets up the project with dependencies needs for local testing/linting

# Name of the temporary directory in which compiled binaries will be stored.
temp_directory="tmp"

binaries=(errcheck gofumpt goimports golint gosec shadow staticcheck golangci-lint)

echo -e "\nInstalling Shadow"
go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow

echo -e "\nInstalling GoImports"
go install golang.org/x/tools/cmd/goimports

echo -e "\nInstalling GoLint"
go install golang.org/x/lint/golint

echo -e "\nInstalling StaticCheck"
go install honnef.co/go/tools/cmd/staticcheck

echo -e "\nInstalling ErrCheck"
go install github.com/kisielk/errcheck

echo -e "\nInstalling GoSec"
go install github.com/securego/gosec/cmd/gosec

echo -e "\nInstalling Golang CI - Lint"
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.34.1

# Fetching the location of the compiled binaries.
gopath=$(go env GOPATH)

# Creating a directory named `tmp` - command will be ignored if the directory exists
mkdir -p "./${temp_directory}"

path=$(pwd)

# Switching over to the path in GoRoot containing the compiled binaries.
cd "${gopath}/bin/"

# Moving all the binaries into the directory.
for file in "${binaries[@]}"; do
  mv "${file}" "${path}/tmp/"
done

echo -e "\n\nExecution completed. Compiled binaries are now stored in \`${path}/${temp_directory}\`\n"
