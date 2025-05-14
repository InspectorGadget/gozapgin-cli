# GoZapGin CLI

## Description
GoZapGin is a command-line tool that allows you to easily create and manage Lambda functions build on top of Gin Framework. It provides a simple Command Line Interface (CLI) to create, build, and deploy your Lambda functions with ease.

[Features roadmap (Board)](https://github.com/users/InspectorGadget/projects/2)

## Features
- Create a new Lambda function with a Gin framework
- Build the Lambda function
- Deploy/Undeploy the Lambda function to/from AWS
- Generate a zip file for the Lambda function

## Usage
1. Download the latest release from the [releases page](https://github.com/InspectorGadget/gozapgin-cli/releases)
2. Copy the binary to your PATH
3. Rename and make the binary executable.
   ```bash
   mv gozap-macos-amd64 /path/go/gozapgin
   chmod +x /path/to/gozapgin
   ```
4. Run the command to test the installation
   ```bash
   gozapgin --help
   ```

## Commands

| Command | Flag | Description |
|---------|------|-------------|
| `gozapgin init` | `--stage` | Initialize GoZapGin project with the specified stage |
| | `--name` | Set the project name |
| | `--bucket` | Specify the deployment bucket |
| `gozapgin deploy` | `--stage` | Deploy the Lambda function to the specified stage |
| `gozapgin undeploy` | `--stage` | Undeploy the Lambda function from the specified stage |

## Examples

```bash
# Initialize a new project
gozapgin init --stage production --name test-project --bucket deploymentbucket

# Deploy to production
gozapgin deploy --stage production

# Undeploy from production
gozapgin undeploy --stage production
```
