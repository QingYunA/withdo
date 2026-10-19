// Command adhd-core is adhd-todo's command line: an MCP server for coding
// agents (mcp) and a plain-text report of what they are doing (overview).
package main

import (
	"os"

	"github.com/QingYunA/adhd-todo/internal/cli"
)

func main() { os.Exit(cli.Main(os.Args[1:])) }
