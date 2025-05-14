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
  user: AdminUser;
};

type AdminMutation = {
  __typename?: 'AdminMutation';
  banUser: BanUserOrErrorPayload;
  unbanUser: UnbanUserOrErrorPayload;
  updateSubgraph: UpdateSubgraphOrErrorPayload;
  updateUserSubgraphAccess: UpdateUserSubgraphAccessOrErrorPayload;
};


type AdminMutationBanUserArgs = {
  input: BanUserInput;
};


type AdminMutationUnbanUserArgs = {
  input: UnbanUserInput;
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
  subgraph?: Maybe<AdminSubgraph>;
  userSubgraphAccess?: Maybe<AdminUserSubgraphAccess>;
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

type AdminSubgraph = {
  __typename?: 'AdminSubgraph';
  color?: Maybe<Scalars['String']['output']>;
  createdAt: Scalars['Time']['output'];
  id: Scalars['Int64']['output'];
  name: Scalars['String']['output'];
};

type AdminSubgraphsConnection = {
  __typename?: 'AdminSubgraphsConnection';
  nodes: Array<AdminSubgraph>;
};

type AdminUser = {
  __typename?: 'AdminUser';
  ban?: Maybe<UserBan>;
  createdAt: Scalars['Time']['output'];
  email: Scalars['String']['output'];
  id: Scalars['Int64']['output'];
};

type AdminUserBansConnection = {
  __typename?: 'AdminUserBansConnection';
  nodes: Array<UserBan>;
};

type AdminUserSubgraphAccess = {
  __typename?: 'AdminUserSubgraphAccess';
  createdAt: Scalars['Time']['output'];
  expiresAt?: Maybe<Scalars['Time']['output']>;
  id: Scalars['Int64']['output'];
  subgraph: AdminSubgraph;
  subgraphId: Scalars['Int64']['output'];
  user: AdminUser;
  userId: Scalars['Int64']['output'];
};

type AdminUserSubgraphAccessesConnection = {
  __typename?: 'AdminUserSubgraphAccessesConnection';
  nodes: Array<AdminUserSubgraphAccess>;
};

type AdminUsersConnection = {
  __typename?: 'AdminUsersConnection';
  nodes: Array<AdminUser>;
};

type BanUserInput = {
  reason: Scalars['String']['input'];
  userId: Scalars['Int64']['input'];
};

type BanUserOrErrorPayload = BanUserPayload | ErrorPayload;

type BanUserPayload = {
  __typename?: 'BanUserPayload';
  user: AdminUser;
  userId: Scalars['Int64']['output'];
};

type CreatePaymentLinkInput = {
  email?: InputMaybe<Scalars['String']['input']>;
  offerId: Scalars['String']['input'];
  paymentType: PaymentType;
  returnPath: Scalars['String']['input'];
};

type CreatePaymentLinkOrErrorPayload = CreatePaymentLinkPayload | ErrorPayload;

type CreatePaymentLinkPayload = {
  __typename?: 'CreatePaymentLinkPayload';
  redirectUrl: Scalars['String']['output'];
  token?: Maybe<Scalars['String']['output']>;
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
  createPaymentLink: CreatePaymentLinkOrErrorPayload;
  requestEmailSignInCode: RequestEmailSignInCodeOrErrorPayload;
  signInByEmail: SignInOrErrorPayload;
  signOut: SignOutOrErrorPayload;
};


type MutationCreatePaymentLinkArgs = {
  input: CreatePaymentLinkInput;
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

type Offer = {
  __typename?: 'Offer';
  id: Scalars['String']['output'];
  priceUSD: Scalars['Float']['output'];
  subgraphs: Array<Subgraph>;
};

enum PaymentType {
  Crypto = 'CRYPTO'
}

type Purchase = {
  __typename?: 'Purchase';
  id: Scalars['String']['output'];
  status: Scalars['String']['output'];
  successful: Scalars['Boolean']['output'];
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
  code: Scalars['String']['input'];
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
  homePath: Scalars['String']['output'];
  name: Scalars['String']['output'];
  offers: Array<Offer>;
};

type UnbanUserInput = {
  userId: Scalars['Int64']['input'];
};

type UnbanUserOrErrorPayload = ErrorPayload | UnbanUserPayload;

type UnbanUserPayload = {
  __typename?: 'UnbanUserPayload';
  user: AdminUser;
  userId: Scalars['Int64']['output'];
};

type UpdateSubgraphInput = {
  color: Scalars['String']['input'];
  id: Scalars['Int64']['input'];
};

type UpdateSubgraphOrErrorPayload = ErrorPayload | UpdateSubgraphPayload;

type UpdateSubgraphPayload = {
  __typename?: 'UpdateSubgraphPayload';
  subgraph: AdminSubgraph;
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
  email: Scalars['String']['output'];
  subgraphAccesses: Array<UserSubgraphAccess>;
};

type UserBan = {
  __typename?: 'UserBan';
  bannedBy?: Maybe<Admin>;
  createdAt: Scalars['Time']['output'];
  reason: Scalars['String']['output'];
  user: AdminUser;
  userId: Scalars['Int64']['output'];
};

type UserSubgraphAccess = {
  __typename?: 'UserSubgraphAccess';
  createdAt: Scalars['Time']['output'];
  expiresAt?: Maybe<Scalars['Time']['output']>;
  id: Scalars['ID']['output'];
  subgraph: Subgraph;
};

type Viewer = {
  __typename?: 'Viewer';
  activePurchases: Array<Purchase>;
  id: Scalars['ID']['output'];
  offers: Array<Offer>;
  user?: Maybe<User>;
};


type ViewerOffersArgs = {
  subgraphs?: InputMaybe<Array<Scalars['String']['input']>>;
};

type AdminUnbanUserMutationVariables = Exact<{
  input: UnbanUserInput;
}>;


type AdminUnbanUserMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'UnbanUserPayload', user: { __typename: 'AdminUser', id: any } } } };

type AdminListNoteViewsQueryVariables = Exact<{ [key: string]: never; }>;


type AdminListNoteViewsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allNoteViews: { __typename?: 'AdminNoteViewsConnection', nodes: Array<{ __typename?: 'NoteView', id: string, path: string, title: string, free: boolean }> } } };

type AdminListSubgraphsQueryVariables = Exact<{ [key: string]: never; }>;


type AdminListSubgraphsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename: 'AdminSubgraph', id: any, name: string, color?: string | null, createdAt: any }> } } };

type AdminListUserBansQueryVariables = Exact<{ [key: string]: never; }>;


type AdminListUserBansQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allUserUserBans: { __typename?: 'AdminUserBansConnection', nodes: Array<{ __typename?: 'UserBan', createdAt: any, reason: string, id: any, user: { __typename: 'AdminUser', email: string }, bannedBy?: { __typename?: 'Admin', user: { __typename?: 'AdminUser', email: string } } | null }> } } };

type AdminListUsersQueryVariables = Exact<{ [key: string]: never; }>;


type AdminListUsersQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allUsers: { __typename?: 'AdminUsersConnection', nodes: Array<{ __typename?: 'AdminUser', id: any, email: string, createdAt: any, ban?: { __typename?: 'UserBan', reason: string } | null }> } } };

type AdminListUserSubgraphAccessesQueryVariables = Exact<{ [key: string]: never; }>;


type AdminListUserSubgraphAccessesQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', data: { __typename?: 'AdminUserSubgraphAccessesConnection', nodes: Array<{ __typename: 'AdminUserSubgraphAccess', id: any, createdAt: any, expiresAt?: any | null, subgraph: { __typename?: 'AdminSubgraph', name: string }, user: { __typename?: 'AdminUser', id: any, email: string } }> } } };

type AdminSelectSubgraphQueryVariables = Exact<{ [key: string]: never; }>;


type AdminSelectSubgraphQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename?: 'AdminSubgraph', id: any, name: string }> } } };

type AdminBanUserMutationVariables = Exact<{
  input: BanUserInput;
}>;


type AdminBanUserMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', banUser: { __typename: 'BanUserPayload', user: { __typename: 'AdminUser', id: any } } | { __typename: 'ErrorPayload', message: string } } };

type AdminNoteViewQueryVariables = Exact<{
  id: Scalars['String']['input'];
}>;


type AdminNoteViewQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', noteView?: { __typename: 'NoteView', path: string, title: string, permalink: string } | null } };

type AdminShowSubgraphQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


type AdminShowSubgraphQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', subgraph?: { __typename?: 'AdminSubgraph', id: any, name: string, color?: string | null } | null } };

type UpdateSubgraphMutationVariables = Exact<{
  input: UpdateSubgraphInput;
}>;


type UpdateSubgraphMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename: 'ErrorPayload', message: string } | { __typename: 'UpdateSubgraphPayload', subgraph: { __typename: 'AdminSubgraph', id: any, color?: string | null } } } };

type AdminUserSubgraphAccessQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


type AdminUserSubgraphAccessQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename?: 'AdminSubgraph', id: any, name: string }> }, userSubgraphAccess?: { __typename?: 'AdminUserSubgraphAccess', userId: any, subgraphId: any, expiresAt?: any | null } | null } };

type AdminUpdateUserSubgraphAccessMutationVariables = Exact<{
  input: UpdateUserSubgraphAccessInput;
}>;


type AdminUpdateUserSubgraphAccessMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'UpdateUserSubgraphAccessPayload', userSubgraphAccess: { __typename: 'UserSubgraphAccess', expiresAt?: any | null } } } };

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

type ViewerQueryVariables = Exact<{ [key: string]: never; }>;


type ViewerQuery = { __typename?: 'Query', viewer: { __typename?: 'Viewer', id: string, user?: { __typename?: 'User', email: string } | null } };

type PaywallActivePurchaseQueryQueryVariables = Exact<{ [key: string]: never; }>;


type PaywallActivePurchaseQueryQuery = { __typename?: 'Query', viewer: { __typename?: 'Viewer', activePurchases: Array<{ __typename?: 'Purchase', id: string, status: string, successful: boolean }> } };

type PaywallQueryQueryVariables = Exact<{
  subgraphs: Array<Scalars['String']['input']> | Scalars['String']['input'];
}>;


type PaywallQueryQuery = { __typename?: 'Query', viewer: { __typename?: 'Viewer', offers: Array<{ __typename?: 'Offer', id: string, priceUSD: number, subgraphs: Array<{ __typename?: 'Subgraph', name: string }> }> } };

type CreatePaymentLinkMutationVariables = Exact<{
  input: CreatePaymentLinkInput;
}>;


type CreatePaymentLinkMutation = { __typename?: 'Mutation', data: { __typename?: 'CreatePaymentLinkPayload', redirectUrl: string } | { __typename?: 'ErrorPayload', message: string } };

type UserSubscriptionsQueryVariables = Exact<{ [key: string]: never; }>;


type UserSubscriptionsQuery = { __typename?: 'Query', viewer: { __typename?: 'Viewer', user?: { __typename?: 'User', subgraphAccesses: Array<{ __typename?: 'UserSubgraphAccess', id: string, createdAt: any, expiresAt?: any | null, subgraph: { __typename?: 'Subgraph', name: string, homePath: string } }> } | null } };

export function $trip2g_graphql_request(query: '\n\t\t\t\tmutation AdminUnbanUser($input: UnbanUserInput!) {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tdata: unbanUser(input: $input) {\n\t\t\t\t\t\t\t... on UnbanUserPayload {\n\t\t\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\n\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t', variables: AdminUnbanUserMutationVariables): AdminUnbanUserMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminListNoteViews {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallNoteViews {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tpath\n\t\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t\t\tfree\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminListNoteViewsQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminListSubgraphs {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallSubgraphs {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\tcolor\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminListSubgraphsQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminListUserBans {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallUserUserBans {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid: userId\n\t\t\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\tbannedBy {\n\t\t\t\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\treason\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminListUserBansQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminListUsers {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallUsers {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\tban { reason }\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminListUsersQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminListUserSubgraphAccesses {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tdata: allUserSubgraphAccesses {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\texpiresAt\n\t\t\t\t\t\t\t\tsubgraph {\n\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminListUserSubgraphAccessesQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminSelectSubgraph {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tallSubgraphs {\n\t\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t'): AdminSelectSubgraphQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tmutation AdminBanUser($input: BanUserInput!) {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tbanUser(input: $input) {\n\t\t\t\t\t\t\t... on BanUserPayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tuser { id, __typename }\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t', variables: AdminBanUserMutationVariables): AdminBanUserMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminNoteView($id: String!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tnoteView(id: $id) {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tpath\n\t\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t\t\tpermalink\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminNoteViewQueryVariables): AdminNoteViewQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminShowSubgraph($id: Int64!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tsubgraph(id: $id) {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\tcolor\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminShowSubgraphQueryVariables): AdminShowSubgraphQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation UpdateSubgraph($input: UpdateSubgraphInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: updateSubgraph(input: $input) {\n\t\t\t\t\t\t\t\t... on UpdateSubgraphPayload {\n\t\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\t\tsubgraph {\n\t\t\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t\tcolor\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: UpdateSubgraphMutationVariables): UpdateSubgraphMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminUserSubgraphAccess($id: Int64!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tallSubgraphs {\n\t\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\n\t\t\t\t\t\t\tuserSubgraphAccess(id: $id) {\n\t\t\t\t\t\t\t\tuserId\n\t\t\t\t\t\t\t\tsubgraphId\n\t\t\t\t\t\t\t\texpiresAt\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminUserSubgraphAccessQueryVariables): AdminUserSubgraphAccessQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminUpdateUserSubgraphAccess($input: UpdateUserSubgraphAccessInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: updateUserSubgraphAccess(input: $input) {\n\t\t\t\t\t\t\t\t... on UpdateUserSubgraphAccessPayload {\n\t\t\t\t\t\t\t\t\tuserSubgraphAccess {\n\t\t\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\t\t\texpiresAt\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminUpdateUserSubgraphAccessMutationVariables): AdminUpdateUserSubgraphAccessMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tmutation SignOut {\n\t\t\t\t\tdata: signOut {\n\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t\t... on SignOutPayload {\n\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\tviewer {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): SignOutMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation RequestEmailSignInCode($input: RequestEmailSignInCodeInput!) {\n\t\t\t\t\t\tdata: requestEmailSignInCode(input: $input) {\n\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t... on RequestEmailSignInCodePayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tsuccess\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: RequestEmailSignInCodeMutationVariables): RequestEmailSignInCodeMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation SignInByEmail($input: SignInByEmailInput!) {\n\t\t\t\t\t\tdata: signInByEmail(input: $input) {\n\t\t\t\t\t\t\t... on SignInPayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\ttoken\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: SignInByEmailMutationVariables): SignInByEmailMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery Viewer {\n\t\t\t\t\tviewer {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): ViewerQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery PaywallActivePurchaseQuery {\n\t\t\t\t\tviewer {\n\t\t\t\t\t\tactivePurchases {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tstatus\n\t\t\t\t\t\t\tsuccessful\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): PaywallActivePurchaseQueryQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery PaywallQuery($subgraphs: [String!]!) {\n\t\t\t\t\tviewer {\n\t\t\t\t\t\toffers(subgraphs: $subgraphs) {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tpriceUSD\n\t\t\t\t\t\t\tsubgraphs {\n\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\n\t\t\t\t}\n\t\t\t', variables: PaywallQueryQueryVariables): PaywallQueryQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tmutation CreatePaymentLink($input: CreatePaymentLinkInput!) {\n\t\t\t\t\tdata: createPaymentLink(input: $input) {\n\t\t\t\t\t\t... on CreatePaymentLinkPayload {\n\t\t\t\t\t\t\tredirectUrl\n\t\t\t\t\t\t}\n\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t\t', variables: CreatePaymentLinkMutationVariables): CreatePaymentLinkMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery UserSubscriptions {\n\t\t\t\t\tviewer {\n\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\tsubgraphAccesses {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\texpiresAt\n\t\t\t\t\t\t\t\tsubgraph {\n\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t\thomePath\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): UserSubscriptionsQuery

export function $trip2g_graphql_request(query: any, variables?: any) { return $trip2g_graphql_raw_request(query, variables); }

export function $trip2g_graphql_subscription(query: any, variables?: any) { return $trip2g_graphql_raw_subscription(query, variables); }



export const $trip2g_graphql_persist_queries = {"AdminUnbanUser":"9512bb945535dd9ef2fe1dacc60073ce01fc8d8f3e93fec6583ab2dc455d1309","AdminListNoteViews":"246778d60afacd4ff9dce34784ee070ce17cc89edaa5a49ed26236ae65b5a159","AdminListSubgraphs":"4e45ae80a24576cab70fdbb4790a0a7acbde1171823623ed8d7a00e495b596cd","AdminListUserBans":"69e1b4b4cb152647fa3474d44d1fccc7e5bfdc9bfa95d60b9726d4e52c94b3b4","AdminListUsers":"6f4fcb27423e59a080c8c0ba8cc8b69628bf9f961bdfbce5c62bf60c19db4075","AdminListUserSubgraphAccesses":"79ca5aafd82b91a579c3ea6232fa7a32773f2b74846785a00e0f0deee9854eca","AdminSelectSubgraph":"a801d4b303ea060e27e22011e2d7cc74c9a17a95e0877622e55e4012640c11a7","AdminBanUser":"b7eaca2436a0420749fd9818239ddaaf03cf39d13fb051af54e1ddfc16b0a376","AdminNoteView":"71287b914ef21187cc102872ffec82d7d3dbe8f1c7583391227f3829c627cc0d","AdminShowSubgraph":"e55ded2d39f46e61be846d0eaa24e9434a4cbc400e4d93fbde65dadeb14dd559","UpdateSubgraph":"a1ed76b00109cb6eac864a2c57d63d956d4c34676eb519e8d0a33abcd8c8945b","AdminUserSubgraphAccess":"a624b706534097050759fc343d7335c36f60b7fd8cb7eac35979d2c2e0b166da","AdminUpdateUserSubgraphAccess":"1f5c1a927c82b86a3d0e0313744fe51f023ed479f955e6846225bff4ab75a91d","SignOut":"8e1a898d776a103a20b7dbdfbeb8196fce0175ffe38fa7af9da2848cefdadedf","RequestEmailSignInCode":"cc5a33407c1a3cca08dbcba6847a88298eec84b10f18449244cf13e458449ef9","SignInByEmail":"5678d2ca29b77529e0b9942845dcea2c941c6b5a234863dc86ac2ffaad668614","Viewer":"b43077267ca739524818d4fb3a825a8eae7bf443c628c5f4270b114ab32977ed","PaywallActivePurchaseQuery":"49eef6eeabb08d2251039c9b3ca67cade0c7af866c29300712400aa44d34d15b","PaywallQuery":"600ed11b093fa49a35a066487efd179edb8593e273565d02b13c87bb04cb49d7","CreatePaymentLink":"3c4ea8c746c573dc9a48a303c13f2b6f7ac3ccad23045f8e989105ea0df98b7a","UserSubscriptions":"1e67f2ab803afe77d6859c2130b2fede9f9edce8c9d841e14ce552c3ecf40115"}

}