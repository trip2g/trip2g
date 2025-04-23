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
  listUsers: AdminUsersConnection;
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
};


type MutationRequestEmailSignInCodeArgs = {
  input?: InputMaybe<RequestEmailSignInCodeInput>;
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

type User = {
  __typename?: 'User';
  createdAt: Scalars['Time']['output'];
  email: Scalars['String']['output'];
  id: Scalars['Int64']['output'];
};

type Viewer = {
  __typename?: 'Viewer';
  id: Scalars['ID']['output'];
  user?: Maybe<User>;
};

type RequestEmailSignInCodeMutationVariables = Exact<{
  input: RequestEmailSignInCodeInput;
}>;


type RequestEmailSignInCodeMutation = { __typename?: 'Mutation', data: { __typename: 'ErrorPayload', message: string } | { __typename: 'RequestEmailSignInCodePayload', success: boolean } };

export function $trip2g_graphql_request(query: '\n\t\t\t\tmutation RequestEmailSignInCode($input: RequestEmailSignInCodeInput!) {\n\t\t\t\t\tdata: requestEmailSignInCode(input: $input) {\n\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t\t... on RequestEmailSignInCodePayload {\n\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\tsuccess\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t', variables: MutationRequestEmailSignInCodeArgs): RequestEmailSignInCodeMutation

export function $trip2g_graphql_request(query: any, variables?: any) { return $trip2g_graphql_raw_request(query, variables); }

}