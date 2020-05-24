package flathttp

import (
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"net"
	"testing"
	"time"
)

func getAvailableAddrs(t testing.TB, scheme string, n int) []string {
	t.Helper()

	lns := make([]net.Listener, 0, n)
	addrs := make([]string, 0, n)
	for i := 0; i < n; i++ {
		ln, err := net.Listen(scheme, ":0")
		require.NoError(t, err)

		lns = append(lns, ln)
		addrs = append(addrs, scheme+"://"+ln.Addr().String())
	}
	for _, ln := range lns {
		require.NoError(t, ln.Close())
	}
	return addrs
}

func TestServer(t *testing.T) {
	defer goleak.VerifyNone(t)

	srv := NewServer(
		&Config{
			Addrs:           getAvailableAddrs(t, "tcp", 1),
			ShutdownTimeout: 100 * time.Millisecond,
		},
	)

	for i := 0; i < 10; i++ {
		require.Empty(t, srv.Start())

		srv.Update(
			&Config{
				Addrs:           append(srv.cfg.Addrs, getAvailableAddrs(t, "tcp", 1)...),
				ShutdownTimeout: 100 * time.Millisecond,
			},
		)
	}

	require.Empty(t, srv.Stop())
}
