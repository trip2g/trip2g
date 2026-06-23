// Minimal mock OIDC issuer for e2e tests of the trip2g OIDC login flow.
//
// Networking split (the app runs inside docker-compose, the browser + this
// server run on the host):
//   - issuer / token / userinfo / jwks  -> reached by the APP container, so they
//     use the `host.docker.internal` host (requires `extra_hosts` on the app
//     service in docker-compose.test.yml).
//   - authorize -> reached by the BROWSER on the host, so it uses `localhost`.
//
// The server binds 0.0.0.0 so the container can reach it via the host gateway.
import http from 'node:http';

export async function startMockIdp({
  port = 19200,
  hostForContainer = 'host.docker.internal',
  hostForBrowser = 'localhost',
} = {}) {
  const issuer = `http://${hostForContainer}:${port}`;

  // Mutable identity returned by /userinfo; set per-test via setUser().
  let currentUser = {
    sub: 'oidc-user-1',
    email: 'oidc@example.com',
    email_verified: true,
    name: 'OIDC User',
    groups: [],
  };

  const json = (res, code, body) => {
    res.writeHead(code, { 'content-type': 'application/json' });
    res.end(JSON.stringify(body));
  };

  const server = http.createServer((req, res) => {
    const url = new URL(req.url, `http://localhost:${port}`);

    if (url.pathname === '/.well-known/openid-configuration') {
      return json(res, 200, {
        issuer,
        authorization_endpoint: `http://${hostForBrowser}:${port}/authorize`,
        token_endpoint: `http://${hostForContainer}:${port}/token`,
        userinfo_endpoint: `http://${hostForContainer}:${port}/userinfo`,
        jwks_uri: `http://${hostForContainer}:${port}/jwks`,
        response_types_supported: ['code'],
        subject_types_supported: ['public'],
        id_token_signing_alg_values_supported: ['RS256'],
        scopes_supported: ['openid', 'email', 'profile', 'groups'],
      });
    }

    // Browser hits this; bounce straight back to the app's callback with a code.
    if (url.pathname === '/authorize') {
      const redirectUri = url.searchParams.get('redirect_uri');
      const state = url.searchParams.get('state') || '';
      const location = `${redirectUri}?code=mock-code&state=${encodeURIComponent(state)}`;
      res.writeHead(302, { location });
      return res.end();
    }

    if (url.pathname === '/token' && req.method === 'POST') {
      return json(res, 200, {
        access_token: 'mock-access-token',
        token_type: 'Bearer',
        expires_in: 3600,
        id_token: 'mock.unsigned.idtoken',
      });
    }

    if (url.pathname === '/userinfo') {
      return json(res, 200, currentUser);
    }

    if (url.pathname === '/jwks') {
      return json(res, 200, { keys: [] });
    }

    res.writeHead(404);
    res.end();
  });

  await new Promise((resolve) => server.listen(port, '0.0.0.0', resolve));

  return {
    issuer,
    browserURL: `http://${hostForBrowser}:${port}`,
    setUser(u) {
      currentUser = { ...currentUser, ...u };
    },
    close() {
      return new Promise((resolve) => server.close(resolve));
    },
  };
}
