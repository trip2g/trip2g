import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 10 }, // Ramp up to 10 users
    { duration: '1m', target: 10 },  // Stay at 10 users
    { duration: '30s', target: 0 },  // Ramp down to 0 users
  ],
};

const BASE_URL = 'http://localhost:8081';
const GRAPHQL_ENDPOINT = `${BASE_URL}/graphql`;
const TARGET_PAGE = '/ponedeljnik_9_iyunya_2025';

// Helper function to make GraphQL requests
function graphqlRequest(query, variables = {}) {
  const payload = JSON.stringify({
    query: query,
    variables: variables,
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Accept': '*/*',
      'Origin': BASE_URL,
      'Referer': BASE_URL,
      'User-Agent': 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1',
    },
  };

  return http.post(GRAPHQL_ENDPOINT, payload, params);
}

// Request email sign-in code
function requestSignInCode(email) {
  const query = `
    mutation RequestEmailSignInCode($input: RequestEmailSignInCodeInput!) {
      data: requestEmailSignInCode(input: $input) {
        ... on ErrorPayload {
          __typename
          message
        }
        ... on RequestEmailSignInCodePayload {
          __typename
          success
        }
      }
    }
  `;

  const variables = {
    input: {
      email: email,
    },
  };

  return graphqlRequest(query, variables);
}

// Sign in with email and code
function signInByEmail(email, code) {
  const query = `
    mutation SignInByEmail($input: SignInByEmailInput!) {
      data: signInByEmail(input: $input) {
        ... on SignInPayload {
          __typename
          token
        }
        ... on ErrorPayload {
          __typename
          message
        }
      }
    }
  `;

  const variables = {
    input: {
      email: email,
      code: code,
    },
  };

  return graphqlRequest(query, variables);
}

// Get current viewer info
function getCurrentViewer() {
  const query = `
    query Viewer {
      viewer {
        id
        user {
          email
        }
      }
    }
  `;

  return graphqlRequest(query, {});
}

export default function () {
  // 50% of users will be guests, 50% will be authenticated
  const isAuthenticated = Math.random() < 0.5;

  if (isAuthenticated) {
    // Authenticated user flow
    console.log('Testing as authenticated user');

    const jar = http.cookieJar();

    // Step 1: Request sign-in code
    const email = 'hello@example.com';
    const requestCodeRes = requestSignInCode(email);

    check(requestCodeRes, {
      'request code status is 200': (r) => r && r.status === 200,
      'request code successful': (r) => {
        if (!r || !r.body) return false;
        try {
          const body = JSON.parse(r.body);
          return body.data && body.data.__typename === 'RequestEmailSignInCodePayload' && body.data.success === true;
        } catch (e) {
          return false;
        }
      },
    });

    // Step 2: Sign in with code
    const code = '111111';
    const signInRes = signInByEmail(email, code);

    check(signInRes, {
      'sign in status is 200': (r) => r && r.status === 200,
      'sign in successful': (r) => {
        if (!r || !r.body) return false;
        try {
          const body = JSON.parse(r.body);
          console.log('signInRes body:', body);
          return body.data && body.data.__typename === 'SignInPayload';
        } catch (e) {
          return false;
        }
      },
    });

    // Step 3: Verify authentication by getting viewer info
    const viewerRes = getCurrentViewer();
    console.log('Viewer response:', viewerRes.body);
    console.log('jar', jar.cookiesForURL(BASE_URL));

    check(viewerRes, {
      'viewer request status is 200': (r) => r && r.status === 200,
      'viewer has user': (r) => {
        if (!r || !r.body) return false;
        try {
          const body = JSON.parse(r.body);
          return body.data && body.data.viewer && body.data.viewer.user && body.data.viewer.user.email === email;
        } catch (e) {
          return false;
        }
      },
    });
  }

  const pageRes = http.get(`${BASE_URL}${TARGET_PAGE}`);

  check(pageRes, {
    'authenticated page access status is 200': (r) => r && r.status === 200,
    'authenticated page loads': (r) => r && r.body && r.body.includes('Проснулся по будильнику на браслете'),
  });

  sleep(2); // Delay between iterations
}
