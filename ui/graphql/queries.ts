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

type Admin = {
  __typename?: 'Admin';
  user: User;
};

type AdminMutation = {
  __typename?: 'AdminMutation';
  updateSubgraph: UpdateSubgraphOrErrorPayload;
  updateUserSubgraphAccess: UpdateUserSubgraphAccessOrErrorPayload;
};


type AdminMutationUpdateSubgraphArgs = {
  input: UpdateSubgraphInput;
};


type AdminMutationUpdateUserSubgraphAccessArgs = {
  input: UpdateUserSubgraphAccessInput;
};

type AdminNoteViewsConnection = {
  __typename?: 'AdminNoteViewsConnection';
  nodes: Array<NoteView>;
};

type AdminQuery = {
  __typename?: 'AdminQuery';
  allNoteViews: AdminNoteViewsConnection;
  allSubgraphs: AdminSubgraphsConnection;
  allUserSubgraphAccesses: AdminUserSubgraphAccessesConnection;
  allUserUserBans: AdminUserBansConnection;
  allUsers: AdminUsersConnection;
  noteView?: Maybe<NoteView>;
  subgraph?: Maybe<Subgraph>;
  userSubgraphAccess?: Maybe<UserSubgraphAccess>;
};


type AdminQueryNoteViewArgs = {
  id: Scalars['String']['input'];
};


type AdminQuerySubgraphArgs = {
  id: Scalars['Int64']['input'];
};


type AdminQueryUserSubgraphAccessArgs = {
  id: Scalars['Int64']['input'];
};

type AdminSubgraphsConnection = {
  __typename?: 'AdminSubgraphsConnection';
  nodes: Array<Subgraph>;
};

type AdminUserBansConnection = {
  __typename?: 'AdminUserBansConnection';
  nodes: Array<UserBan>;
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
  admin: AdminMutation;
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

type NoteView = {
  __typename?: 'NoteView';
  content: Scalars['String']['output'];
  free: Scalars['Boolean']['output'];
  html: Scalars['String']['output'];
  id: Scalars['String']['output'];
  path: Scalars['String']['output'];
  permalink: Scalars['String']['output'];
  title: Scalars['String']['output'];
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

type Subgraph = {
  __typename?: 'Subgraph';
  color?: Maybe<Scalars['String']['output']>;
  createdAt: Scalars['Time']['output'];
  id: Scalars['Int64']['output'];
  name: Scalars['String']['output'];
};

type UpdateSubgraphInput = {
  color: Scalars['String']['input'];
  id: Scalars['Int64']['input'];
};

type UpdateSubgraphOrErrorPayload = ErrorPayload | UpdateSubgraphPayload;

type UpdateSubgraphPayload = {
  __typename?: 'UpdateSubgraphPayload';
  subgraph: Subgraph;
};

type UpdateUserSubgraphAccessInput = {
  expiresAt?: InputMaybe<Scalars['Time']['input']>;
  id: Scalars['Int64']['input'];
  subgraphId?: InputMaybe<Scalars['Int64']['input']>;
};

type UpdateUserSubgraphAccessOrErrorPayload = ErrorPayload | UpdateUserSubgraphAccessPayload;

type UpdateUserSubgraphAccessPayload = {
  __typename?: 'UpdateUserSubgraphAccessPayload';
  userSubgraphAccess: UserSubgraphAccess;
};

type User = {
  __typename?: 'User';
  createdAt: Scalars['Time']['output'];
  email: Scalars['String']['output'];
  id: Scalars['Int64']['output'];
};

type UserBan = {
  __typename?: 'UserBan';
  bannedBy?: Maybe<Admin>;
  createdAt: Scalars['Time']['output'];
  reason: Scalars['String']['output'];
  user: User;
  userId: Scalars['Int64']['output'];
};

type UserSubgraphAccess = {
  __typename?: 'UserSubgraphAccess';
  createdAt: Scalars['Time']['output'];
  expiresAt?: Maybe<Scalars['Time']['output']>;
  id: Scalars['Int64']['output'];
  subgraph: Subgraph;
  subgraphId: Scalars['Int64']['output'];
  user: User;
  userId: Scalars['Int64']['output'];
};

type Viewer = {
  __typename?: 'Viewer';
  id: Scalars['ID']['output'];
  user?: Maybe<User>;
};

type AdminListNoteViewsQueryVariables = Exact<{ [key: string]: never; }>;


type AdminListNoteViewsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allNoteViews: { __typename?: 'AdminNoteViewsConnection', nodes: Array<{ __typename?: 'NoteView', id: string, path: string, title: string, free: boolean }> } } };

type AdminListSubgraphsQueryVariables = Exact<{ [key: string]: never; }>;


type AdminListSubgraphsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename: 'Subgraph', id: any, name: string, color?: string | null, createdAt: any }> } } };

type AdminListUserBansQueryVariables = Exact<{ [key: string]: never; }>;


type AdminListUserBansQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allUserUserBans: { __typename?: 'AdminUserBansConnection', nodes: Array<{ __typename?: 'UserBan', createdAt: any, reason: string, id: any, user: { __typename?: 'User', email: string }, bannedBy?: { __typename?: 'Admin', user: { __typename?: 'User', email: string } } | null }> } } };

type AdminListUsersQueryVariables = Exact<{ [key: string]: never; }>;


type AdminListUsersQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allUsers: { __typename?: 'AdminUsersConnection', nodes: Array<{ __typename?: 'User', id: any, email: string, createdAt: any }> } } };

type AdminListUserSubgraphAccessesQueryVariables = Exact<{ [key: string]: never; }>;


type AdminListUserSubgraphAccessesQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', data: { __typename?: 'AdminUserSubgraphAccessesConnection', nodes: Array<{ __typename: 'UserSubgraphAccess', id: any, createdAt: any, expiresAt?: any | null, subgraph: { __typename?: 'Subgraph', name: string }, user: { __typename?: 'User', id: any, email: string } }> } } };

type AdminSelectSubgraphQueryVariables = Exact<{ [key: string]: never; }>;


type AdminSelectSubgraphQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename?: 'Subgraph', id: any, name: string }> } } };

type AdminNoteViewQueryVariables = Exact<{
  id: Scalars['String']['input'];
}>;


type AdminNoteViewQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', noteView?: { __typename: 'NoteView', path: string, title: string, permalink: string } | null } };

type AdminShowSubgraphQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


type AdminShowSubgraphQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', subgraph?: { __typename?: 'Subgraph', id: any, name: string, color?: string | null } | null } };

type UpdateSubgraphMutationVariables = Exact<{
  input: UpdateSubgraphInput;
}>;


type UpdateSubgraphMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename: 'ErrorPayload', message: string } | { __typename: 'UpdateSubgraphPayload', subgraph: { __typename: 'Subgraph', id: any, color?: string | null } } } };

type AdminUserSubgraphAccessQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


type AdminUserSubgraphAccessQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename?: 'Subgraph', id: any, name: string }> }, userSubgraphAccess?: { __typename?: 'UserSubgraphAccess', userId: any, subgraphId: any, expiresAt?: any | null } | null } };

type AdminUpdateUserSubgraphAccessMutationVariables = Exact<{
  input: UpdateUserSubgraphAccessInput;
}>;


type AdminUpdateUserSubgraphAccessMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'UpdateUserSubgraphAccessPayload', userSubgraphAccess: { __typename: 'UserSubgraphAccess', expiresAt?: any | null } } } };

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

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminListNoteViews {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallNoteViews {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tpath\n\t\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t\t\tfree\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminListNoteViewsQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminListSubgraphs {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallSubgraphs {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\tcolor\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminListSubgraphsQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminListUserBans {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallUserUserBans {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid: userId\n\t\t\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\tbannedBy {\n\t\t\t\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\treason\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminListUserBansQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminListUsers {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallUsers {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminListUsersQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminListUserSubgraphAccesses {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tdata: allUserSubgraphAccesses {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\texpiresAt\n\t\t\t\t\t\t\t\tsubgraph {\n\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminListUserSubgraphAccessesQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminSelectSubgraph {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tallSubgraphs {\n\t\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t'): AdminSelectSubgraphQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminNoteView($id: String!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tnoteView(id: $id) {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tpath\n\t\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t\t\tpermalink\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminNoteViewQueryVariables): AdminNoteViewQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminShowSubgraph($id: Int64!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tsubgraph(id: $id) {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\tcolor\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminShowSubgraphQueryVariables): AdminShowSubgraphQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation UpdateSubgraph($input: UpdateSubgraphInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: updateSubgraph(input: $input) {\n\t\t\t\t\t\t\t\t... on UpdateSubgraphPayload {\n\t\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\t\tsubgraph {\n\t\t\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t\tcolor\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: UpdateSubgraphMutationVariables): UpdateSubgraphMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminUserSubgraphAccess($id: Int64!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tallSubgraphs {\n\t\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\n\t\t\t\t\t\t\tuserSubgraphAccess(id: $id) {\n\t\t\t\t\t\t\t\tuserId\n\t\t\t\t\t\t\t\tsubgraphId\n\t\t\t\t\t\t\t\texpiresAt\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminUserSubgraphAccessQueryVariables): AdminUserSubgraphAccessQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminUpdateUserSubgraphAccess($input: UpdateUserSubgraphAccessInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: updateUserSubgraphAccess(input: $input) {\n\t\t\t\t\t\t\t\t... on UpdateUserSubgraphAccessPayload {\n\t\t\t\t\t\t\t\t\tuserSubgraphAccess {\n\t\t\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\t\t\texpiresAt\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminUpdateUserSubgraphAccessMutationVariables): AdminUpdateUserSubgraphAccessMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery Viewer {\n\t\t\t\t\tviewer {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): ViewerQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tmutation SignOut {\n\t\t\t\t\tdata: signOut {\n\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t\t... on SignOutPayload {\n\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\tviewer {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): SignOutMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation RequestEmailSignInCode($input: RequestEmailSignInCodeInput!) {\n\t\t\t\t\t\tdata: requestEmailSignInCode(input: $input) {\n\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t... on RequestEmailSignInCodePayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tsuccess\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: RequestEmailSignInCodeMutationVariables): RequestEmailSignInCodeMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation SignInByEmail($input: SignInByEmailInput!) {\n\t\t\t\t\t\tdata: signInByEmail(input: $input) {\n\t\t\t\t\t\t\t... on SignInPayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\ttoken\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: SignInByEmailMutationVariables): SignInByEmailMutation

export function $trip2g_graphql_request(query: any, variables?: any) { return $trip2g_graphql_raw_request(query, variables); }

export const $trip2g_graphql_persist_queries = {"AdminListNoteViews":"1fa4c73890f89fe68dfe9785f82d77f91e0a323c21b6ecc44a4b7b1c79eceb72","AdminListSubgraphs":"2cee87730b8d154a6683eee863b7d55859a5cb0d8bcac5d8a6eb3ad699ef7b10","AdminListUserBans":"05184810ff7c0b6308695a9f9d2913cb6dc8ade814c1cdc8bc66a52600a6bf12","AdminListUsers":"bb65e89c75590f0431371bbf80d782707ac4ace59860bffaff808863ccec219c","AdminListUserSubgraphAccesses":"c401571acb47ee9727398b1e47a53d2e864b12882ba31004a51c783dee6ac689","AdminSelectSubgraph":"2f46ae2f709229d7c3dd5a0c564863f57129f0500f1c4ec14f7086ada407d1fe","AdminNoteView":"30bfefe9ac3f3a4c70bd46e1866026909bf4ac013c8c274f52d560336d834068","AdminShowSubgraph":"a85d72ae054c667afe00190423dbc3420cf0fab11cd7842d37f1646ba8fccebe","UpdateSubgraph":"910386c507a3405267c729bc1359fd439a0c89726092eca906fbb72e7da3546b","AdminUserSubgraphAccess":"46d844334f0f79f0f0a79d6cad4a80721d8345976c91744d3c754b23ce3a8f52","AdminUpdateUserSubgraphAccess":"66152953987b3f528e2f0899561dfa2db2a5fc13d6ac152808b0c0eaa573e379","Viewer":"3a8c636e5b92b0bd849229fbc238e1bd5d9a6ab0f00b9726cfcd06ad7464a685","SignOut":"cba3ef4c85f3b6d17dcd080aef3350968b170a4d2bfe6bd82dad23587984c3e4","RequestEmailSignInCode":"d36752898539b45c7fcb5a54a4bed0aa246ca0373cbb7604fde55977e7a7d406","SignInByEmail":"dcd280c068ca17383a6a9bd60d6f9f448c57bb96fc52f6b02d6c3fb1c32c518f"}

}