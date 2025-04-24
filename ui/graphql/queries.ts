namespace $.$$ {


type Maybe<T> = T | null;
type InputMaybe<T> = Maybe<T>;
type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
type MakeEmpty<T extends { [key: string]: unknown }, K extends keyof T> = { [_ in K]?: never };
type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
/** All built-in and custom scalars, mapped to their actual values */
type Scalars = {
  ID: { input: string; output: string; }
  String: { input: string; output: string; }
  Boolean: { input: boolean; output: boolean; }
  Int: { input: number; output: number; }
  Float: { input: number; output: number; }
  Int64: { input: any; output: any; }
  Time: { input: any; output: any; }
};

type AdminQuery = {
  __typename?: 'AdminQuery';
  allUserSubgraphAccesses: AdminUserSubgraphAccessesConnection;
  allUsers: AdminUsersConnection;
};

type AdminUserSubgraphAccessesConnection = {
  __typename?: 'AdminUserSubgraphAccessesConnection';
  nodes: Array<UserSubgraphAccess>;
};

type AdminUsersConnection = {
  __typename?: 'AdminUsersConnection';
  nodes: Array<User>;
};

type ErrorPayload = {
  __typename?: 'ErrorPayload';
  byFields: Array<FieldMessage>;
  message: Scalars['String']['output'];
};

type FieldMessage = {
  __typename?: 'FieldMessage';
  name: Scalars['String']['output'];
  value: Scalars['String']['output'];
};

type Mutation = {
  __typename?: 'Mutation';
  requestEmailSignInCode: RequestEmailSignInCodeOrErrorPayload;
  signInByEmail: SignInOrErrorPayload;
  signOut: SignOutOrErrorPayload;
};


type MutationRequestEmailSignInCodeArgs = {
  input: RequestEmailSignInCodeInput;
};


type MutationSignInByEmailArgs = {
  input: SignInByEmailInput;
};

type Query = {
  __typename?: 'Query';
  admin: AdminQuery;
  viewer: Viewer;
};

type RequestEmailSignInCodeInput = {
  email: Scalars['String']['input'];
};

type RequestEmailSignInCodeOrErrorPayload = ErrorPayload | RequestEmailSignInCodePayload;

type RequestEmailSignInCodePayload = {
  __typename?: 'RequestEmailSignInCodePayload';
  success: Scalars['Boolean']['output'];
};

type SignInByEmailInput = {
  code: Scalars['Int']['input'];
  email: Scalars['String']['input'];
};

type SignInOrErrorPayload = ErrorPayload | SignInPayload;

type SignInPayload = {
  __typename?: 'SignInPayload';
  token: Scalars['String']['output'];
  viewer: Viewer;
};

type SignOutOrErrorPayload = ErrorPayload | SignOutPayload;

type SignOutPayload = {
  __typename?: 'SignOutPayload';
  viewer: Viewer;
};

type User = {
  __typename?: 'User';
  createdAt: Scalars['Time']['output'];
  email: Scalars['String']['output'];
  id: Scalars['Int64']['output'];
};

type UserSubgraphAccess = {
  __typename?: 'UserSubgraphAccess';
  createdAt: Scalars['Time']['output'];
  expiresAt?: Maybe<Scalars['Time']['output']>;
  id: Scalars['Int64']['output'];
  subgraphID: Scalars['Int64']['output'];
  userID: Scalars['Int64']['output'];
};

type Viewer = {
  __typename?: 'Viewer';
  id: Scalars['ID']['output'];
  user?: Maybe<User>;
};

type AdminListUsersQueryVariables = Exact<{ [key: string]: never; }>;


type AdminListUsersQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allUsers: { __typename?: 'AdminUsersConnection', nodes: Array<{ __typename?: 'User', id: any, email: string, createdAt: any }> } } };

type AdminListUserSubgraphAccessesQueryVariables = Exact<{ [key: string]: never; }>;


type AdminListUserSubgraphAccessesQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', data: { __typename?: 'AdminUserSubgraphAccessesConnection', nodes: Array<{ __typename?: 'UserSubgraphAccess', id: any, createdAt: any, expiresAt?: any | null }> } } };

type ViewerQueryVariables = Exact<{ [key: string]: never; }>;


type ViewerQuery = { __typename?: 'Query', viewer: { __typename?: 'Viewer', id: string, user?: { __typename?: 'User', id: any, email: string, createdAt: any } | null } };

type SignOutMutationVariables = Exact<{ [key: string]: never; }>;


type SignOutMutation = { __typename?: 'Mutation', data: { __typename: 'ErrorPayload', message: string } | { __typename: 'SignOutPayload', viewer: { __typename?: 'Viewer', id: string } } };

type RequestEmailSignInCodeMutationVariables = Exact<{
  input: RequestEmailSignInCodeInput;
}>;


type RequestEmailSignInCodeMutation = { __typename?: 'Mutation', data: { __typename: 'ErrorPayload', message: string } | { __typename: 'RequestEmailSignInCodePayload', success: boolean } };

type SignInByEmailMutationVariables = Exact<{
  input: SignInByEmailInput;
}>;


type SignInByEmailMutation = { __typename?: 'Mutation', data: { __typename: 'ErrorPayload', message: string } | { __typename: 'SignInPayload', token: string } };

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminListUsers {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallUsers {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminListUsersQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminListUserSubgraphAccesses {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tdata: allUserSubgraphAccesses {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\texpiresAt\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminListUserSubgraphAccessesQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery Viewer {\n\t\t\t\t\tviewer {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): ViewerQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tmutation SignOut {\n\t\t\t\t\tdata: signOut {\n\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t\t... on SignOutPayload {\n\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\tviewer {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): SignOutMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation RequestEmailSignInCode($input: RequestEmailSignInCodeInput!) {\n\t\t\t\t\t\tdata: requestEmailSignInCode(input: $input) {\n\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t... on RequestEmailSignInCodePayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tsuccess\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: RequestEmailSignInCodeMutationVariables): RequestEmailSignInCodeMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation SignInByEmail($input: SignInByEmailInput!) {\n\t\t\t\t\t\tdata: signInByEmail(input: $input) {\n\t\t\t\t\t\t\t... on SignInPayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\ttoken\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: SignInByEmailMutationVariables): SignInByEmailMutation

export function $trip2g_graphql_request(query: any, variables?: any) { return $trip2g_graphql_raw_request(query, variables); }

}