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

type Query = {
  __typename?: 'Query';
  admin: AdminQuery;
};

type User = {
  __typename?: 'User';
  createdAt: Scalars['Time']['output'];
  email: Scalars['String']['output'];
  id: Scalars['Int64']['output'];
};

type RequestEmailSigninCodeQueryVariables = Exact<{ [key: string]: never; }>;


type RequestEmailSigninCodeQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', listUsers: { __typename?: 'AdminUsersConnection', nodes: Array<{ __typename?: 'User', id: any }> } } };

	export const $trip2g_graphql_request_email_signin_code = (variables: RequestEmailSigninCodeQueryVariables) =>
		$trip2g_graphql_request<RequestEmailSigninCodeQuery>(`query RequestEmailSigninCode {
  admin {
    listUsers {
      nodes {
        id
      }
    }
  }
}`, variables)

}