package cli

import "os"

// ParseArgs returns the config file path from CLI arguments.
//
// It accepts both `-c <path>` and `--config <path>`. If no explicit config
// flag is provided, it falls back to `config.yaml`.
func ParseArgs(argv []string) string {
	for i := 0; i < len(argv); i++ {
		if (argv[i] == "-c" || argv[i] == "--config") && i+1 < len(argv) {
			return argv[i+1]
		}
	}
	return "config.yaml"
}

// Exit terminates the process with the given exit code.
func Exit(code int) {
	os.Exit(code)
}
