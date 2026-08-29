package testcore

import (
	"sync"
	"testing"
	"time"

	"go.temporal.io/server/common/headers"
	"go.temporal.io/server/common/testing/parallelsuite"
	"go.temporal.io/server/common/testing/testcontext"
	"google.golang.org/grpc/metadata"
)

type TestEnvSuite struct {
	parallelsuite.Suite[*TestEnvSuite]
}

func TestTestEnvSuite(t *testing.T) {
	parallelsuite.Run(t, &TestEnvSuite{})
}

func (s *TestEnvSuite) TestDedicatedClusterGuard_NoErrorWithoutExplicitRequest() {
	guard := newDedicatedClusterGuard(false)

	s.NoError(guard.validate())
}

func (s *TestEnvSuite) TestDedicatedClusterGuard_FailsWhenUnused() {
	guard := newDedicatedClusterGuard(true)

	s.EqualError(guard.validate(),
		`testcore.WithDedicatedCluster() was requested but no dedicated-cluster-only feature was used`)
}

func (s *TestEnvSuite) TestDedicatedClusterGuard_NoErrorAfterUse() {
	guard := newDedicatedClusterGuard(true)
	guard.record("global hook")

	s.NoError(guard.validate())
}

func (s *TestEnvSuite) TestDedicatedClusterGuard_ConcurrentRecord() {
	guard := newDedicatedClusterGuard(true)
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			guard.record("reason")
		})
	}
	wg.Wait()
	s.NoError(guard.validate())
}

func (s *TestEnvSuite) TestContextCachesVersionHeadersAcrossExtension() {
	env := NewEnv(s.T())
	ctx := env.Context()
	md, ok := metadata.FromOutgoingContext(ctx)
	s.Require().True(ok)
	s.Require().Equal([]string{headers.ClientNameServer}, md.Get(headers.ClientNameHeaderName))
	s.Require().Equal([]string{headers.ServerVersion}, md.Get(headers.ClientVersionHeaderName))

	testcontext.EnsureRemaining(ctx, s.T(), testcontext.DefaultTimeout()+time.Second)
	s.Require().Same(ctx, env.Context())
}
