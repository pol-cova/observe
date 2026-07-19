package observe

import (
	"fmt"

	"github.com/pol-cova/observe/internal/server"
	"github.com/spf13/cobra"
)

var (
	servePort  int
	serveBind  string
	serveToken string
	servePath  string
	serveCORS  string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Expose machine info as a JSON HTTP endpoint",
	Long:  "Start a read-only HTTP server that returns the same JSON snapshot as `observe snapshot`.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return server.Listen(server.Options{
			Addr:  fmt.Sprintf("%s:%d", serveBind, servePort),
			Path:  servePath,
			Token: serveToken,
			CORS:  serveCORS,
		})
	},
}

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 8080, "port to listen on")
	serveCmd.Flags().StringVar(&serveBind, "bind", "127.0.0.1", "address to bind (use 0.0.0.0 for external access)")
	serveCmd.Flags().StringVar(&serveToken, "token", "", "optional bearer token required for requests")
	serveCmd.Flags().StringVar(&servePath, "path", "/info", "URL path for the info endpoint")
	serveCmd.Flags().StringVar(&serveCORS, "cors", "*", "Access-Control-Allow-Origin header value")
}
