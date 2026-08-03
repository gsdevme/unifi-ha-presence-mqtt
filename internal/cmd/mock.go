package cmd

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/gsdevme/unifi-ha-presence-mqtt/internal/mock"
)

var mockAddr string

var mockCmd = &cobra.Command{
	Use:   "mock",
	Short: "Run the standalone UniFi controller mock",
	Long: "Serves canned active-client responses so the full pipeline can run without " +
		"a real UniFi controller. Run serve with MODE=mock (set MOCK_URL if this server " +
		"is not on the default :8090). Lists phone1 and phone2 as active by default.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		logger := newLogger("info", "text")
		opts := mock.Defaults()
		opts.Logger = logger
		srv := mock.New(opts)

		logger.Info("mock UniFi controller listening", "addr", mockAddr)
		httpSrv := &http.Server{Addr: mockAddr, Handler: srv.Handler()}

		serveErr := make(chan error, 1)
		go func() {
			if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				serveErr <- err
				return
			}
			serveErr <- nil
		}()

		select {
		case err := <-serveErr:
			return err
		case <-cmd.Context().Done():
			logger.Info("shutting down mock")
			if err := shutdownServer(httpSrv); err != nil {
				return fmt.Errorf("shutdown mock: %w", err)
			}
			// Drain the serve goroutine so a serve error racing the signal isn't
			// discarded; it always sends exactly once.
			return <-serveErr
		}
	},
}

func init() {
	mockCmd.Flags().StringVar(&mockAddr, "addr", ":8090", "listen address for the mock controller")
}
