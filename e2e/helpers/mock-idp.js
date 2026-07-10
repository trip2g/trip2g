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
//
// id_token is a real RS256 JWT, signed with a key generated at startup and
// published via /jwks, so the backend's signature/iss/aud/exp/nonce/sub
// verification (see internal/oidcauth) passes against this mock.
import http from 'node:http';
import crypto from 'node:crypto';

// base64url without padding, as used for both JWK members and JWT segments.
function base64url(input) {
  const buf = Buffer.isBuffer(input) ? input : Buffer.from(input);
  return buf.toString('base64').replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

export async function startMockIdp({
  port = 19200,
  hostForContainer = 'host.docker.internal',
  hostForBrowser = 'localhost',
} = {}) {
  const issuer = `http://${hostForContainer}:${port}`;
  const clientId = 'trip2g-test';
  const kid = 'mock-key-1';

  const { publicKey, privateKey } = crypto.generateKeyPairSync('rsa', { modulusLength: 2048 });
  const jwk = publicKey.export({ format: 'jwk' });

  // nonce from the authorize request, keyed by the code we hand back so the
  // token endpoint can echo it into the id_token.
  const noncesByCode = new Map();

  // Mutable identity returned by /userinfo; set per-test via setUser().
  let currentUser = {
    sub: 'oidc-user-1',
    email: 'oidc@example.com',
    email_verified: true,
    name: 'OIDC User',
    groups: [],
  };
  let currentIDTokenClaims = {};

  function signIdToken(nonce) {
    const header = { alg: 'RS256', typ: 'JWT', kid };
    const now = Math.floor(Date.now() / 1000);
    const payload = {
      iss: issuer,
      aud: clientId,
      sub: currentUser.sub,
      exp: now + 3600,
      iat: now,
      ...(nonce ? { nonce } : {}),
      ...currentIDTokenClaims,
    };
    const signingInput = `${base64url(JSON.stringify(header))}.${base64url(JSON.stringify(payload))}`;
    const signature = crypto.createSign('RSA-SHA256').update(signingInput).sign(privateKey);
    return `${signingInput}.${base64url(signature)}`;
  }

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
      const nonce = url.searchParams.get('nonce') || '';
      const code = 'mock-code';
      noncesByCode.set(code, nonce);
      const location = `${redirectUri}?code=${code}&state=${encodeURIComponent(state)}`;
      res.writeHead(302, { location });
      return res.end();
    }

    if (url.pathname === '/token' && req.method === 'POST') {
      // Body parsing isn't needed: the mock only ever hands out one code at a
      // time (tests run serially), so just use the last stored nonce.
      const code = 'mock-code';
      const nonce = noncesByCode.get(code) || '';
      return json(res, 200, {
        access_token: 'mock-access-token',
        token_type: 'Bearer',
        expires_in: 3600,
        id_token: signIdToken(nonce),
      });
    }

    if (url.pathname === '/userinfo') {
      return json(res, 200, currentUser);
    }

    if (url.pathname === '/jwks') {
      return json(res, 200, { keys: [{ ...jwk, kid, use: 'sig', alg: 'RS256' }] });
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
      currentIDTokenClaims = {};
    },
    setIdTokenClaims(claims) {
      currentIDTokenClaims = { ...claims };
    },
    close() {
      return new Promise((resolve) => server.close(resolve));
    },
  };
}
