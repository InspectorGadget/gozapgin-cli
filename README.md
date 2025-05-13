# GoZapGin CLI

## Description
GoZapGin is a command-line tool that allows you to easily create and manage Lambda functions build on top of Gin Framework. It provides a simple Command Line Interface (CLI) to create, build, and deploy your Lambda functions with ease.

## Features
- Create a new Lambda function with a Gin framework
- Build the Lambda function
- Deploy the Lambda function to AWS
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