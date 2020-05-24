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

	addrs := make([]string, 0, n)
	for i := 0; i < n; i++ {
		ln, err := net.Listen(scheme, ":0")
		require.NoError(t, err)
		require.NoError(t, ln.Close())

		addrs = append(addrs, scheme+"://"+ln.Addr().String())
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
