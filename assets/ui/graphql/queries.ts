namespace $.$$ {


export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
export type MakeEmpty<T extends { [key: string]: unknown }, K extends keyof T> = { [_ in K]?: never };
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: { input: string; output: string; }
  String: { input: string; output: string; }
  Boolean: { input: boolean; output: boolean; }
  Int: { input: number; output: number; }
  Float: { input: number; output: number; }
  Int64: { input: any; output: any; }
  Time: { input: any; output: any; }
  Upload: { input: any; output: any; }
};

export type Admin = {
  __typename?: 'Admin';
  grantedAt: Scalars['Time']['output'];
  grantedBy?: Maybe<AdminUser>;
  id: Scalars['Int64']['output'];
  user: AdminUser;
};

export type AdminAdminsConnection = {
  __typename?: 'AdminAdminsConnection';
  nodes: Array<Admin>;
};

export type AdminApiKey = {
  __typename?: 'AdminApiKey';
  createdAt: Scalars['Time']['output'];
  createdBy: AdminUser;
  description: Scalars['String']['output'];
  disabledAt?: Maybe<Scalars['Time']['output']>;
  disabledBy?: Maybe<AdminUser>;
  id: Scalars['Int64']['output'];
};

export type AdminApiKeyLog = {
  __typename?: 'AdminApiKeyLog';
  actionName: Scalars['String']['output'];
  createdAt: Scalars['Time']['output'];
  ip: Scalars['String']['output'];
};

export type AdminApiKeyLogsConnection = {
  __typename?: 'AdminApiKeyLogsConnection';
  nodes: Array<AdminApiKeyLog>;
};

export type AdminApiKeysConnection = {
  __typename?: 'AdminApiKeysConnection';
  nodes: Array<AdminApiKey>;
};

export type AdminLatestNoteViewsConnection = {
  __typename?: 'AdminLatestNoteViewsConnection';
  nodes: Array<NoteView>;
};

export type AdminMutation = {
  __typename?: 'AdminMutation';
  banUser: BanUserOrErrorPayload;
  createApiKey: CreateApiKeyOrErrorPayload;
  createOffer: CreateOfferOrErrorPayload;
  createRelease: CreateReleaseOrErrorPayload;
  disableApiKey: DisableApiKeyOrErrorPayload;
  makeReleaseLive: MakeReleaseLiveOrErrorPayload;
  unbanUser: UnbanUserOrErrorPayload;
  updateNoteGraphPositions: UpdateNoteGraphPositionsOrErrorPayload;
  updateOffer: UpdateOfferOrErrorPayload;
  updateSubgraph: UpdateSubgraphOrErrorPayload;
  updateUserSubgraphAccess: UpdateUserSubgraphAccessOrErrorPayload;
};


export type AdminMutationBanUserArgs = {
  input: BanUserInput;
};


export type AdminMutationCreateApiKeyArgs = {
  input: CreateApiKeyInput;
};


export type AdminMutationCreateOfferArgs = {
  input: CreateOfferInput;
};


export type AdminMutationCreateReleaseArgs = {
  input: CreateReleaseInput;
};


export type AdminMutationDisableApiKeyArgs = {
  input: DisableApiKeyInput;
};


export type AdminMutationMakeReleaseLiveArgs = {
  input: MakeReleaseLiveInput;
};


export type AdminMutationUnbanUserArgs = {
  input: UnbanUserInput;
};


export type AdminMutationUpdateNoteGraphPositionsArgs = {
  input: UpdateNoteGraphPositionsInput;
};


export type AdminMutationUpdateOfferArgs = {
  input: UpdateOfferInput;
};


export type AdminMutationUpdateSubgraphArgs = {
  input: UpdateSubgraphInput;
};


export type AdminMutationUpdateUserSubgraphAccessArgs = {
  input: UpdateUserSubgraphAccessInput;
};

export type AdminOffer = {
  __typename?: 'AdminOffer';
  createdAt: Scalars['Time']['output'];
  endsAt?: Maybe<Scalars['Time']['output']>;
  id: Scalars['Int64']['output'];
  lifetime?: Maybe<Scalars['String']['output']>;
  priceUSD: Scalars['Float']['output'];
  publicId: Scalars['String']['output'];
  startsAt?: Maybe<Scalars['Time']['output']>;
  subgraphIds: Array<Scalars['Int64']['output']>;
  subgraphs: Array<AdminSubgraph>;
};

export type AdminOffersConnection = {
  __typename?: 'AdminOffersConnection';
  nodes: Array<AdminOffer>;
};

export type AdminPurchase = {
  __typename?: 'AdminPurchase';
  createdAt: Scalars['Time']['output'];
  email: Scalars['String']['output'];
  id: Scalars['String']['output'];
  offer: AdminOffer;
  offerId: Scalars['Int64']['output'];
  paymentProvider: Scalars['String']['output'];
  status: Scalars['String']['output'];
  successful: Scalars['Boolean']['output'];
  user?: Maybe<AdminUser>;
  userId?: Maybe<Scalars['Int64']['output']>;
};

export type AdminPurchasesConnection = {
  __typename?: 'AdminPurchasesConnection';
  nodes: Array<AdminPurchase>;
};

export type AdminQuery = {
  __typename?: 'AdminQuery';
  allAdmins: AdminAdminsConnection;
  allApiKeys: AdminApiKeysConnection;
  allLatestNoteViews: AdminLatestNoteViewsConnection;
  allOffers: AdminOffersConnection;
  allPurchases: AdminPurchasesConnection;
  allReleases: AdminReleasesConnection;
  allSubgraphs: AdminSubgraphsConnection;
  allUserSubgraphAccesses: AdminUserSubgraphAccessesConnection;
  allUserUserBans: AdminUserBansConnection;
  allUsers: AdminUsersConnection;
  apiKeyLogs: AdminApiKeyLogsConnection;
  noteView?: Maybe<NoteView>;
  offer?: Maybe<AdminOffer>;
  purchase?: Maybe<AdminPurchase>;
  subgraph?: Maybe<AdminSubgraph>;
  userSubgraphAccess?: Maybe<AdminUserSubgraphAccess>;
};


export type AdminQueryApiKeyLogsArgs = {
  filter: ApiKeyLogsFilterInput;
};


export type AdminQueryNoteViewArgs = {
  id: Scalars['String']['input'];
};


export type AdminQueryOfferArgs = {
  id: Scalars['Int64']['input'];
};


export type AdminQueryPurchaseArgs = {
  id: Scalars['String']['input'];
};


export type AdminQuerySubgraphArgs = {
  id: Scalars['Int64']['input'];
};


export type AdminQueryUserSubgraphAccessArgs = {
  id: Scalars['Int64']['input'];
};

export type AdminRelease = {
  __typename?: 'AdminRelease';
  createdAt: Scalars['Time']['output'];
  createdBy: AdminUser;
  homeNote?: Maybe<NoteView>;
  homeNoteVersionId?: Maybe<Scalars['Int64']['output']>;
  id: Scalars['Int64']['output'];
  isLive: Scalars['Boolean']['output'];
  title: Scalars['String']['output'];
};

export type AdminReleasesConnection = {
  __typename?: 'AdminReleasesConnection';
  nodes: Array<AdminRelease>;
};

export type AdminSubgraph = {
  __typename?: 'AdminSubgraph';
  color?: Maybe<Scalars['String']['output']>;
  createdAt: Scalars['Time']['output'];
  id: Scalars['Int64']['output'];
  name: Scalars['String']['output'];
};

export type AdminSubgraphsConnection = {
  __typename?: 'AdminSubgraphsConnection';
  nodes: Array<AdminSubgraph>;
};

export type AdminUser = {
  __typename?: 'AdminUser';
  ban?: Maybe<UserBan>;
  createdAt: Scalars['Time']['output'];
  email: Scalars['String']['output'];
  id: Scalars['Int64']['output'];
};

export type AdminUserBansConnection = {
  __typename?: 'AdminUserBansConnection';
  nodes: Array<UserBan>;
};

export type AdminUserSubgraphAccess = {
  __typename?: 'AdminUserSubgraphAccess';
  createdAt: Scalars['Time']['output'];
  expiresAt?: Maybe<Scalars['Time']['output']>;
  id: Scalars['Int64']['output'];
  subgraph: AdminSubgraph;
  subgraphId: Scalars['Int64']['output'];
  user: AdminUser;
  userId: Scalars['Int64']['output'];
};

export type AdminUserSubgraphAccessesConnection = {
  __typename?: 'AdminUserSubgraphAccessesConnection';
  nodes: Array<AdminUserSubgraphAccess>;
};

export type AdminUsersConnection = {
  __typename?: 'AdminUsersConnection';
  nodes: Array<AdminUser>;
};

export type ApiKeyLogsFilterInput = {
  apiKeyId?: InputMaybe<Scalars['Int64']['input']>;
};

export type BanUserInput = {
  reason: Scalars['String']['input'];
  userId: Scalars['Int64']['input'];
};

export type BanUserOrErrorPayload = BanUserPayload | ErrorPayload;

export type BanUserPayload = {
  __typename?: 'BanUserPayload';
  user: AdminUser;
  userId: Scalars['Int64']['output'];
};

export type CreateApiKeyInput = {
  description: Scalars['String']['input'];
};

export type CreateApiKeyOrErrorPayload = CreateApiKeyPayload | ErrorPayload;

export type CreateApiKeyPayload = {
  __typename?: 'CreateApiKeyPayload';
  apiKey: AdminApiKey;
  value: Scalars['String']['output'];
};

export type CreateOfferInput = {
  endsAt?: InputMaybe<Scalars['Time']['input']>;
  lifetime?: InputMaybe<Scalars['String']['input']>;
  priceUSD: Scalars['Float']['input'];
  startsAt?: InputMaybe<Scalars['Time']['input']>;
  subgraphIds: Array<Scalars['Int64']['input']>;
};

export type CreateOfferOrErrorPayload = CreateOfferPayload | ErrorPayload;

export type CreateOfferPayload = {
  __typename?: 'CreateOfferPayload';
  offer: AdminOffer;
};

export type CreatePaymentLinkInput = {
  email?: InputMaybe<Scalars['String']['input']>;
  offerId: Scalars['String']['input'];
  paymentType: PaymentType;
  returnPath: Scalars['String']['input'];
};

export type CreatePaymentLinkOrErrorPayload = CreatePaymentLinkPayload | ErrorPayload;

export type CreatePaymentLinkPayload = {
  __typename?: 'CreatePaymentLinkPayload';
  redirectUrl: Scalars['String']['output'];
  token?: Maybe<Scalars['String']['output']>;
};

export type CreateReleaseInput = {
  homeNoteVersionId?: InputMaybe<Scalars['Int64']['input']>;
  title: Scalars['String']['input'];
};

export type CreateReleaseOrErrorPayload = CreateReleasePayload | ErrorPayload;

export type CreateReleasePayload = {
  __typename?: 'CreateReleasePayload';
  release: AdminRelease;
};

export type DisableApiKeyInput = {
  id: Scalars['Int64']['input'];
};

export type DisableApiKeyOrErrorPayload = DisableApiKeyPayload | ErrorPayload;

export type DisableApiKeyPayload = {
  __typename?: 'DisableApiKeyPayload';
  apiKey: AdminApiKey;
};

export type ErrorPayload = {
  __typename?: 'ErrorPayload';
  byFields: Array<FieldMessage>;
  message: Scalars['String']['output'];
};

export type FieldMessage = {
  __typename?: 'FieldMessage';
  name: Scalars['String']['output'];
  value: Scalars['String']['output'];
};

export type MakeReleaseLiveInput = {
  id: Scalars['Int64']['input'];
};

export type MakeReleaseLiveOrErrorPayload = ErrorPayload | MakeReleaseLivePayload;

export type MakeReleaseLivePayload = {
  __typename?: 'MakeReleaseLivePayload';
  release: AdminRelease;
};

export type Mutation = {
  __typename?: 'Mutation';
  admin: AdminMutation;
  createPaymentLink: CreatePaymentLinkOrErrorPayload;
  pushNotes: PushNotesOrErrorPayload;
  requestEmailSignInCode: RequestEmailSignInCodeOrErrorPayload;
  signInByEmail: SignInOrErrorPayload;
  signOut: SignOutOrErrorPayload;
  uploadNoteAsset: UploadNoteAssetOrErrorPayload;
};


export type MutationCreatePaymentLinkArgs = {
  input: CreatePaymentLinkInput;
};


export type MutationPushNotesArgs = {
  input: PushNotesInput;
};


export type MutationRequestEmailSignInCodeArgs = {
  input: RequestEmailSignInCodeInput;
};


export type MutationSignInByEmailArgs = {
  input: SignInByEmailInput;
};


export type MutationUploadNoteAssetArgs = {
  input: UploadNoteAssetInput;
};

export type NotePath = {
  __typename?: 'NotePath';
  latestContentHash: Scalars['String']['output'];
  value: Scalars['String']['output'];
};

export type NoteView = {
  __typename?: 'NoteView';
  content: Scalars['String']['output'];
  free: Scalars['Boolean']['output'];
  graphPosition?: Maybe<Vector2>;
  html: Scalars['String']['output'];
  id: Scalars['String']['output'];
  inLinks: Array<NoteView>;
  isHomePage: Scalars['Boolean']['output'];
  path: Scalars['String']['output'];
  pathId: Scalars['Int64']['output'];
  permalink: Scalars['String']['output'];
  subgraphNames: Array<Scalars['String']['output']>;
  title: Scalars['String']['output'];
  versionId: Scalars['Int64']['output'];
};

export type Offer = {
  __typename?: 'Offer';
  id: Scalars['String']['output'];
  priceUSD: Scalars['Float']['output'];
  subgraphs: Array<Subgraph>;
};

export enum PaymentType {
  Crypto = 'CRYPTO'
}

export type Purchase = {
  __typename?: 'Purchase';
  id: Scalars['String']['output'];
  status: Scalars['String']['output'];
  successful: Scalars['Boolean']['output'];
};

export type PushNoteInput = {
  content: Scalars['String']['input'];
  path: Scalars['String']['input'];
};

export type PushNotesInput = {
  updates: Array<PushNoteInput>;
};

export type PushNotesOrErrorPayload = ErrorPayload | PushNotesPayload;

export type PushNotesPayload = {
  __typename?: 'PushNotesPayload';
  notes: Array<PushedNote>;
};

export type PushedNote = {
  __typename?: 'PushedNote';
  assets: Array<PushedNoteAsset>;
  id: Scalars['Int64']['output'];
  path: Scalars['String']['output'];
};

export type PushedNoteAsset = {
  __typename?: 'PushedNoteAsset';
  path: Scalars['String']['output'];
  sha256Hash?: Maybe<Scalars['String']['output']>;
};

export type Query = {
  __typename?: 'Query';
  admin: AdminQuery;
  notePaths: Array<NotePath>;
  viewer: Viewer;
};

export type RequestEmailSignInCodeInput = {
  email: Scalars['String']['input'];
};

export type RequestEmailSignInCodeOrErrorPayload = ErrorPayload | RequestEmailSignInCodePayload;

export type RequestEmailSignInCodePayload = {
  __typename?: 'RequestEmailSignInCodePayload';
  success: Scalars['Boolean']['output'];
};

export enum Role {
  Admin = 'ADMIN',
  Guest = 'GUEST',
  User = 'USER'
}

export type SignInByEmailInput = {
  code: Scalars['String']['input'];
  email: Scalars['String']['input'];
};

export type SignInOrErrorPayload = ErrorPayload | SignInPayload;

export type SignInPayload = {
  __typename?: 'SignInPayload';
  token: Scalars['String']['output'];
  viewer: Viewer;
};

export type SignOutOrErrorPayload = ErrorPayload | SignOutPayload;

export type SignOutPayload = {
  __typename?: 'SignOutPayload';
  viewer: Viewer;
};

export type Subgraph = {
  __typename?: 'Subgraph';
  homePath: Scalars['String']['output'];
  name: Scalars['String']['output'];
  offers: Array<Offer>;
};

export type UnbanUserInput = {
  userId: Scalars['Int64']['input'];
};

export type UnbanUserOrErrorPayload = ErrorPayload | UnbanUserPayload;

export type UnbanUserPayload = {
  __typename?: 'UnbanUserPayload';
  user: AdminUser;
  userId: Scalars['Int64']['output'];
};

export type UpdateNoteGraphPositionInput = {
  pathId: Scalars['Int64']['input'];
  x: Scalars['Float']['input'];
  y: Scalars['Float']['input'];
};

export type UpdateNoteGraphPositionsInput = {
  positions: Array<UpdateNoteGraphPositionInput>;
};

export type UpdateNoteGraphPositionsOrErrorPayload = ErrorPayload | UpdateNoteGraphPositionsPayload;

export type UpdateNoteGraphPositionsPayload = {
  __typename?: 'UpdateNoteGraphPositionsPayload';
  success: Scalars['Boolean']['output'];
  updatedNoteViews: Array<NoteView>;
};

export type UpdateOfferInput = {
  endsAt?: InputMaybe<Scalars['Time']['input']>;
  id: Scalars['Int64']['input'];
  lifetime?: InputMaybe<Scalars['String']['input']>;
  priceUSD?: InputMaybe<Scalars['Float']['input']>;
  startsAt?: InputMaybe<Scalars['Time']['input']>;
  subgraphIds?: InputMaybe<Array<Scalars['Int64']['input']>>;
};

export type UpdateOfferOrErrorPayload = ErrorPayload | UpdateOfferPayload;

export type UpdateOfferPayload = {
  __typename?: 'UpdateOfferPayload';
  offer: AdminOffer;
};

export type UpdateSubgraphInput = {
  color: Scalars['String']['input'];
  id: Scalars['Int64']['input'];
};

export type UpdateSubgraphOrErrorPayload = ErrorPayload | UpdateSubgraphPayload;

export type UpdateSubgraphPayload = {
  __typename?: 'UpdateSubgraphPayload';
  subgraph: AdminSubgraph;
};

export type UpdateUserSubgraphAccessInput = {
  expiresAt?: InputMaybe<Scalars['Time']['input']>;
  id: Scalars['Int64']['input'];
  subgraphId?: InputMaybe<Scalars['Int64']['input']>;
};

export type UpdateUserSubgraphAccessOrErrorPayload = ErrorPayload | UpdateUserSubgraphAccessPayload;

export type UpdateUserSubgraphAccessPayload = {
  __typename?: 'UpdateUserSubgraphAccessPayload';
  userSubgraphAccess: UserSubgraphAccess;
};

export type UploadNoteAssetInput = {
  absolutePath: Scalars['String']['input'];
  file: Scalars['Upload']['input'];
  noteId: Scalars['Int64']['input'];
  path: Scalars['String']['input'];
  sha256Hash: Scalars['String']['input'];
};

export type UploadNoteAssetOrErrorPayload = ErrorPayload | UploadNoteAssetPayload;

export type UploadNoteAssetPayload = {
  __typename?: 'UploadNoteAssetPayload';
  uploadSkipped: Scalars['Boolean']['output'];
};

export type User = {
  __typename?: 'User';
  email: Scalars['String']['output'];
  subgraphAccesses: Array<UserSubgraphAccess>;
};

export type UserBan = {
  __typename?: 'UserBan';
  bannedBy?: Maybe<Admin>;
  createdAt: Scalars['Time']['output'];
  reason: Scalars['String']['output'];
  user: AdminUser;
  userId: Scalars['Int64']['output'];
};

export type UserSubgraphAccess = {
  __typename?: 'UserSubgraphAccess';
  createdAt: Scalars['Time']['output'];
  expiresAt?: Maybe<Scalars['Time']['output']>;
  id: Scalars['ID']['output'];
  subgraph: Subgraph;
};

export type Vector2 = {
  __typename?: 'Vector2';
  x: Scalars['Float']['output'];
  y: Scalars['Float']['output'];
};

export type Viewer = {
  __typename?: 'Viewer';
  activePurchases: Array<Purchase>;
  id: Scalars['ID']['output'];
  offers: Array<Offer>;
  role: Role;
  user?: Maybe<User>;
};


export type ViewerOffersArgs = {
  subgraphs?: InputMaybe<Array<Scalars['String']['input']>>;
};

export type AdminsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allAdmins: { __typename?: 'AdminAdminsConnection', nodes: Array<{ __typename?: 'Admin', id: any, grantedAt: any, user: { __typename?: 'AdminUser', email: string } }> } } };

export type DisableApiKeyMutationVariables = Exact<{
  input: DisableApiKeyInput;
}>;


export type DisableApiKeyMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'DisableApiKeyPayload', apiKey: { __typename?: 'AdminApiKey', id: any } } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminListApiKeysQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminListApiKeysQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allApiKeys: { __typename?: 'AdminApiKeysConnection', nodes: Array<{ __typename?: 'AdminApiKey', id: any, createdAt: any, description: string, disabledAt?: any | null, createdBy: { __typename?: 'AdminUser', id: any, email: string }, disabledBy?: { __typename?: 'AdminUser', id: any, email: string } | null }> } } };

export type AdminCreateApiKeyMutationVariables = Exact<{
  input: CreateApiKeyInput;
}>;


export type AdminCreateApiKeyMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'CreateApiKeyPayload', value: string, apiKey: { __typename?: 'AdminApiKey', id: any } } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminApiKeyShowQueryQueryVariables = Exact<{
  filter: ApiKeyLogsFilterInput;
}>;


export type AdminApiKeyShowQueryQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', apiKeyLogs: { __typename?: 'AdminApiKeyLogsConnection', nodes: Array<{ __typename?: 'AdminApiKeyLog', createdAt: any, actionName: string, ip: string }> } } };

export type AdminUnbanUserMutationVariables = Exact<{
  input: UnbanUserInput;
}>;


export type AdminUnbanUserMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'UnbanUserPayload', user: { __typename: 'AdminUser', id: any } } } };

export type AdminListNoteViewsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminListNoteViewsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allLatestNoteViews: { __typename?: 'AdminLatestNoteViewsConnection', nodes: Array<{ __typename?: 'NoteView', id: string, path: string, title: string, free: boolean }> } } };

export type AdminListSubgraphsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminListSubgraphsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename: 'AdminSubgraph', id: any, name: string, color?: string | null, createdAt: any }> } } };

export type AdminListUserBansQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminListUserBansQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allUserUserBans: { __typename?: 'AdminUserBansConnection', nodes: Array<{ __typename?: 'UserBan', createdAt: any, reason: string, id: any, user: { __typename: 'AdminUser', email: string }, bannedBy?: { __typename?: 'Admin', user: { __typename?: 'AdminUser', email: string } } | null }> } } };

export type AdminListUsersQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminListUsersQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allUsers: { __typename?: 'AdminUsersConnection', nodes: Array<{ __typename?: 'AdminUser', id: any, email: string, createdAt: any, ban?: { __typename?: 'UserBan', reason: string } | null }> } } };

export type AdminListUserSubgraphAccessesQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminListUserSubgraphAccessesQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', data: { __typename?: 'AdminUserSubgraphAccessesConnection', nodes: Array<{ __typename: 'AdminUserSubgraphAccess', id: any, createdAt: any, expiresAt?: any | null, subgraph: { __typename?: 'AdminSubgraph', name: string }, user: { __typename?: 'AdminUser', id: any, email: string } }> } } };

export type AdminGraphQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminGraphQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename?: 'AdminSubgraph', name: string, color?: string | null }> }, allLatestNoteViews: { __typename?: 'AdminLatestNoteViewsConnection', nodes: Array<{ __typename?: 'NoteView', id: string, subgraphNames: Array<string>, pathId: any, free: boolean, isHomePage: boolean, graphPosition?: { __typename?: 'Vector2', x: number, y: number } | null, inLinks: Array<{ __typename?: 'NoteView', title: string, pathId: any, id: string }> }> } } };

export type AdminUpdateNoteGraphPositionsMutationVariables = Exact<{
  input: UpdateNoteGraphPositionsInput;
}>;


export type AdminUpdateNoteGraphPositionsMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'UpdateNoteGraphPositionsPayload', success: boolean, updatedNoteViews: Array<{ __typename?: 'NoteView', id: string, pathId: any, title: string }> } } };

export type AdminSelectNoteViewQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminSelectNoteViewQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allLatestNoteViews: { __typename?: 'AdminLatestNoteViewsConnection', nodes: Array<{ __typename?: 'NoteView', versionId: any, path: string, title: string }> } } };

export type AdminOffersQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminOffersQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allOffers: { __typename?: 'AdminOffersConnection', nodes: Array<{ __typename?: 'AdminOffer', id: any, publicId: string, createdAt: any, lifetime?: string | null, priceUSD: number, startsAt?: any | null, endsAt?: any | null, subgraphs: Array<{ __typename?: 'AdminSubgraph', name: string }> }> } } };

export type AdminCreateOfferMutationMutationVariables = Exact<{
  input: CreateOfferInput;
}>;


export type AdminCreateOfferMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'CreateOfferPayload', offer: { __typename?: 'AdminOffer', id: any, publicId: string } } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminShowOfferQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminShowOfferQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', offer?: { __typename?: 'AdminOffer', id: any, publicId: string, createdAt: any, lifetime?: string | null, priceUSD: number, startsAt?: any | null, endsAt?: any | null, subgraphIds: Array<any>, subgraphs: Array<{ __typename?: 'AdminSubgraph', id: any, name: string }> } | null } };

export type AdminUpdateOfferMutationMutationVariables = Exact<{
  input: UpdateOfferInput;
}>;


export type AdminUpdateOfferMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'UpdateOfferPayload', offer: { __typename?: 'AdminOffer', id: any, publicId: string } } } };

export type AdminPurchasesQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminPurchasesQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allPurchases: { __typename?: 'AdminPurchasesConnection', nodes: Array<{ __typename?: 'AdminPurchase', id: string, createdAt: any, paymentProvider: string, status: string, successful: boolean, offerId: any, email: string }> } } };

export type AdminMakeReleaseLiveMutationVariables = Exact<{
  input: MakeReleaseLiveInput;
}>;


export type AdminMakeReleaseLiveMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'MakeReleaseLivePayload', release: { __typename?: 'AdminRelease', id: any } } } };

export type AdminReleasesQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminReleasesQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allReleases: { __typename?: 'AdminReleasesConnection', nodes: Array<{ __typename?: 'AdminRelease', id: any, createdAt: any, title: string, isLive: boolean, createdBy: { __typename?: 'AdminUser', email: string } }> } } };

export type AdminCreateReleaseMutationVariables = Exact<{
  input: CreateReleaseInput;
}>;


export type AdminCreateReleaseMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'CreateReleasePayload', release: { __typename?: 'AdminRelease', id: any } } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminBanUserMutationVariables = Exact<{
  input: BanUserInput;
}>;


export type AdminBanUserMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', banUser: { __typename: 'BanUserPayload', user: { __typename: 'AdminUser', id: any } } | { __typename: 'ErrorPayload', message: string } } };

export type AdminNoteViewQueryVariables = Exact<{
  id: Scalars['String']['input'];
}>;


export type AdminNoteViewQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', noteView?: { __typename: 'NoteView', path: string, title: string, permalink: string } | null } };

export type AdminShowSubgraphQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminShowSubgraphQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', subgraph?: { __typename?: 'AdminSubgraph', id: any, name: string, color?: string | null } | null } };

export type UpdateSubgraphMutationVariables = Exact<{
  input: UpdateSubgraphInput;
}>;


export type UpdateSubgraphMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename: 'ErrorPayload', message: string } | { __typename: 'UpdateSubgraphPayload', subgraph: { __typename: 'AdminSubgraph', id: any, color?: string | null } } } };

export type AdminUserSubgraphAccessQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminUserSubgraphAccessQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename?: 'AdminSubgraph', id: any, name: string }> }, userSubgraphAccess?: { __typename?: 'AdminUserSubgraphAccess', userId: any, subgraphId: any, expiresAt?: any | null } | null } };

export type AdminUpdateUserSubgraphAccessMutationVariables = Exact<{
  input: UpdateUserSubgraphAccessInput;
}>;


export type AdminUpdateUserSubgraphAccessMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'UpdateUserSubgraphAccessPayload', userSubgraphAccess: { __typename: 'UserSubgraphAccess', expiresAt?: any | null } } } };

export type AdminSelectSubgraphListQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminSelectSubgraphListQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename?: 'AdminSubgraph', id: any, name: string }> } } };

export type AdminSelectSubgraphQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminSelectSubgraphQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename?: 'AdminSubgraph', id: any, name: string }> } } };

export type SignOutMutationVariables = Exact<{ [key: string]: never; }>;


export type SignOutMutation = { __typename?: 'Mutation', data: { __typename: 'ErrorPayload', message: string } | { __typename: 'SignOutPayload', viewer: { __typename?: 'Viewer', id: string } } };

export type RequestEmailSignInCodeMutationVariables = Exact<{
  input: RequestEmailSignInCodeInput;
}>;


export type RequestEmailSignInCodeMutation = { __typename?: 'Mutation', data: { __typename: 'ErrorPayload', message: string } | { __typename: 'RequestEmailSignInCodePayload', success: boolean } };

export type SignInByEmailMutationVariables = Exact<{
  input: SignInByEmailInput;
}>;


export type SignInByEmailMutation = { __typename?: 'Mutation', data: { __typename: 'ErrorPayload', message: string } | { __typename: 'SignInPayload', token: string } };

export type ViewerQueryVariables = Exact<{ [key: string]: never; }>;


export type ViewerQuery = { __typename?: 'Query', viewer: { __typename?: 'Viewer', id: string, role: Role, user?: { __typename?: 'User', email: string } | null } };

export type PaywallActivePurchaseQueryQueryVariables = Exact<{ [key: string]: never; }>;


export type PaywallActivePurchaseQueryQuery = { __typename?: 'Query', viewer: { __typename?: 'Viewer', activePurchases: Array<{ __typename?: 'Purchase', id: string, status: string, successful: boolean }> } };

export type PaywallQueryQueryVariables = Exact<{
  subgraphs: Array<Scalars['String']['input']> | Scalars['String']['input'];
}>;


export type PaywallQueryQuery = { __typename?: 'Query', viewer: { __typename?: 'Viewer', offers: Array<{ __typename?: 'Offer', id: string, priceUSD: number, subgraphs: Array<{ __typename?: 'Subgraph', name: string }> }> } };

export type CreatePaymentLinkMutationVariables = Exact<{
  input: CreatePaymentLinkInput;
}>;


export type CreatePaymentLinkMutation = { __typename?: 'Mutation', data: { __typename?: 'CreatePaymentLinkPayload', redirectUrl: string } | { __typename?: 'ErrorPayload', message: string } };

export type UserSubscriptionsQueryVariables = Exact<{ [key: string]: never; }>;


export type UserSubscriptionsQuery = { __typename?: 'Query', viewer: { __typename?: 'Viewer', user?: { __typename?: 'User', subgraphAccesses: Array<{ __typename?: 'UserSubgraphAccess', id: string, createdAt: any, expiresAt?: any | null, subgraph: { __typename?: 'Subgraph', name: string, homePath: string } }> } | null } };

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery Admins {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallAdmins {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tgrantedAt\n\t\t\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminsQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation DisableApiKey($input: DisableApiKeyInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: disableApiKey(input: $input) {\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on DisableApiKeyPayload {\n\t\t\t\t\t\t\t\t\tapiKey {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: DisableApiKeyMutationVariables): DisableApiKeyMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminListApiKeys {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallApiKeys {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\tdescription\n\t\t\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\tdisabledAt\n\t\t\t\t\t\t\t\tdisabledBy {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminListApiKeysQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminCreateApiKey($input: CreateApiKeyInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: createApiKey(input: $input) {\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on CreateApiKeyPayload {\n\t\t\t\t\t\t\t\t\tvalue\n\t\t\t\t\t\t\t\t\tapiKey {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminCreateApiKeyMutationVariables): AdminCreateApiKeyMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminApiKeyShowQuery($filter: ApiKeyLogsFilterInput!) {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tapiKeyLogs(filter: $filter) {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\tactionName\n\t\t\t\t\t\t\t\tip\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t', variables: AdminApiKeyShowQueryQueryVariables): AdminApiKeyShowQueryQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tmutation AdminUnbanUser($input: UnbanUserInput!) {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tdata: unbanUser(input: $input) {\n\t\t\t\t\t\t\t... on UnbanUserPayload {\n\t\t\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\n\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t', variables: AdminUnbanUserMutationVariables): AdminUnbanUserMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminListNoteViews {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallLatestNoteViews {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tpath\n\t\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t\t\tfree\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminListNoteViewsQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminListSubgraphs {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallSubgraphs {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\tcolor\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminListSubgraphsQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminListUserBans {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallUserUserBans {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid: userId\n\t\t\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\tbannedBy {\n\t\t\t\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\treason\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminListUserBansQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminListUsers {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallUsers {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\tban { reason }\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminListUsersQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminListUserSubgraphAccesses {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tdata: allUserSubgraphAccesses {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\texpiresAt\n\t\t\t\t\t\t\t\tsubgraph {\n\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminListUserSubgraphAccessesQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminGraph {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallSubgraphs {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\tcolor\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t\tallLatestNoteViews {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tsubgraphNames\n\t\t\t\t\t\t\t\tpathId\n\t\t\t\t\t\t\t\tfree\n\t\t\t\t\t\t\t\tisHomePage\n\t\t\t\t\t\t\t\tgraphPosition{\n\t\t\t\t\t\t\t\t\tx,\n\t\t\t\t\t\t\t\t\ty,\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\tinLinks {\n\t\t\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t\t\t\tpathId\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminGraphQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminUpdateNoteGraphPositions($input: UpdateNoteGraphPositionsInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: updateNoteGraphPositions(input: $input) {\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on UpdateNoteGraphPositionsPayload {\n\t\t\t\t\t\t\t\t\tsuccess\n\t\t\t\t\t\t\t\t\tupdatedNoteViews {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t\tpathId\n\t\t\t\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminUpdateNoteGraphPositionsMutationVariables): AdminUpdateNoteGraphPositionsMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminSelectNoteView {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tallLatestNoteViews {\n\t\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t\tversionId\n\t\t\t\t\t\t\t\t\tpath\n\t\t\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t'): AdminSelectNoteViewQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminOffers {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallOffers {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tpublicId\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\tlifetime\n\t\t\t\t\t\t\t\tpriceUSD\n\t\t\t\t\t\t\t\tstartsAt\n\t\t\t\t\t\t\t\tendsAt\n\t\t\t\t\t\t\t\tsubgraphs {\n\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminOffersQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminCreateOfferMutation($input: CreateOfferInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: createOffer(input: $input) {\n\t\t\t\t\t\t\t\t... on CreateOfferPayload {\n\t\t\t\t\t\t\t\t\toffer {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t\tpublicId\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminCreateOfferMutationMutationVariables): AdminCreateOfferMutationMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminShowOffer($id: Int64!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\toffer(id: $id) {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tpublicId\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\tlifetime\n\t\t\t\t\t\t\t\tpriceUSD\n\t\t\t\t\t\t\t\tstartsAt\n\t\t\t\t\t\t\t\tendsAt\n\t\t\t\t\t\t\t\tsubgraphIds\n\t\t\t\t\t\t\t\tsubgraphs {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminShowOfferQueryVariables): AdminShowOfferQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminUpdateOfferMutation($input: UpdateOfferInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: updateOffer(input: $input) {\n\t\t\t\t\t\t\t\t... on UpdateOfferPayload {\n\t\t\t\t\t\t\t\t\toffer {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t\tpublicId\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminUpdateOfferMutationMutationVariables): AdminUpdateOfferMutationMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminPurchases {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallPurchases {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\tpaymentProvider\n\t\t\t\t\t\t\t\tstatus\n\t\t\t\t\t\t\t\tsuccessful\n\t\t\t\t\t\t\t\tofferId\n\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminPurchasesQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminMakeReleaseLive($input: MakeReleaseLiveInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: makeReleaseLive(input:$input) {\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on MakeReleaseLivePayload {\n\t\t\t\t\t\t\t\t\trelease {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminMakeReleaseLiveMutationVariables): AdminMakeReleaseLiveMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminReleases {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallReleases {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\tcreatedBy{\n\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t\t\tisLive\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminReleasesQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminCreateRelease($input: CreateReleaseInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: createRelease(input: $input) {\n\t\t\t\t\t\t\t\t... on CreateReleasePayload {\n\t\t\t\t\t\t\t\t\trelease {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminCreateReleaseMutationVariables): AdminCreateReleaseMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tmutation AdminBanUser($input: BanUserInput!) {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tbanUser(input: $input) {\n\t\t\t\t\t\t\t... on BanUserPayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tuser { id, __typename }\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t', variables: AdminBanUserMutationVariables): AdminBanUserMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminNoteView($id: String!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tnoteView(id: $id) {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tpath\n\t\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t\t\tpermalink\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminNoteViewQueryVariables): AdminNoteViewQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminShowSubgraph($id: Int64!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tsubgraph(id: $id) {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\tcolor\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminShowSubgraphQueryVariables): AdminShowSubgraphQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation UpdateSubgraph($input: UpdateSubgraphInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: updateSubgraph(input: $input) {\n\t\t\t\t\t\t\t\t... on UpdateSubgraphPayload {\n\t\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\t\tsubgraph {\n\t\t\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t\tcolor\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: UpdateSubgraphMutationVariables): UpdateSubgraphMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminUserSubgraphAccess($id: Int64!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tallSubgraphs {\n\t\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\n\t\t\t\t\t\t\tuserSubgraphAccess(id: $id) {\n\t\t\t\t\t\t\t\tuserId\n\t\t\t\t\t\t\t\tsubgraphId\n\t\t\t\t\t\t\t\texpiresAt\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminUserSubgraphAccessQueryVariables): AdminUserSubgraphAccessQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminUpdateUserSubgraphAccess($input: UpdateUserSubgraphAccessInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: updateUserSubgraphAccess(input: $input) {\n\t\t\t\t\t\t\t\t... on UpdateUserSubgraphAccessPayload {\n\t\t\t\t\t\t\t\t\tuserSubgraphAccess {\n\t\t\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\t\t\texpiresAt\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminUpdateUserSubgraphAccessMutationVariables): AdminUpdateUserSubgraphAccessMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminSelectSubgraphList {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tallSubgraphs {\n\t\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t'): AdminSelectSubgraphListQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminSelectSubgraph {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tallSubgraphs {\n\t\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t'): AdminSelectSubgraphQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tmutation SignOut {\n\t\t\t\t\tdata: signOut {\n\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t\t... on SignOutPayload {\n\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\tviewer {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): SignOutMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation RequestEmailSignInCode($input: RequestEmailSignInCodeInput!) {\n\t\t\t\t\t\tdata: requestEmailSignInCode(input: $input) {\n\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t... on RequestEmailSignInCodePayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tsuccess\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: RequestEmailSignInCodeMutationVariables): RequestEmailSignInCodeMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation SignInByEmail($input: SignInByEmailInput!) {\n\t\t\t\t\t\tdata: signInByEmail(input: $input) {\n\t\t\t\t\t\t\t... on SignInPayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\ttoken\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: SignInByEmailMutationVariables): SignInByEmailMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery Viewer {\n\t\t\t\t\tviewer {\n\t\t\t\t\t\tid\n\t\t\t\t\t\trole\n\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): ViewerQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery PaywallActivePurchaseQuery {\n\t\t\t\t\tviewer {\n\t\t\t\t\t\tactivePurchases {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tstatus\n\t\t\t\t\t\t\tsuccessful\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): PaywallActivePurchaseQueryQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery PaywallQuery($subgraphs: [String!]!) {\n\t\t\t\t\tviewer {\n\t\t\t\t\t\toffers(subgraphs: $subgraphs) {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tpriceUSD\n\t\t\t\t\t\t\tsubgraphs {\n\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\n\t\t\t\t}\n\t\t\t', variables: PaywallQueryQueryVariables): PaywallQueryQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tmutation CreatePaymentLink($input: CreatePaymentLinkInput!) {\n\t\t\t\t\tdata: createPaymentLink(input: $input) {\n\t\t\t\t\t\t... on CreatePaymentLinkPayload {\n\t\t\t\t\t\t\tredirectUrl\n\t\t\t\t\t\t}\n\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t\t', variables: CreatePaymentLinkMutationVariables): CreatePaymentLinkMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery UserSubscriptions {\n\t\t\t\t\tviewer {\n\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\tsubgraphAccesses {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\texpiresAt\n\t\t\t\t\t\t\t\tsubgraph {\n\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t\thomePath\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): UserSubscriptionsQuery

export function $trip2g_graphql_request(query: any, variables?: any) { return $trip2g_graphql_raw_request(query, variables); }

export function $trip2g_graphql_subscription(query: any, variables?: any) { return $trip2g_graphql_raw_subscription(query, variables); }



export const $trip2g_graphql_persist_queries = {"Admins":"7b9f99a6b0b785b43488198eb4dec442d88e4abda7c516677a0611d76878d904","DisableApiKey":"a5852655edeb09cf7db15b0a196cc53cc005b3f8ef7fe7d7d3cdc032087b8b6b","AdminListApiKeys":"1baa27852f59c95f35fe5f35635ad6a617fca6312411bd6cae7cd377d681a52d","AdminCreateApiKey":"c9c10cfb6fa133ac380870427e9b9cdebbc1e9e9b92e6a6313c97f52b537d66f","AdminApiKeyShowQuery":"2f8b56fce14a35ac51fd315e919a352048a118b036a0368f2a66a85c7156c24b","AdminUnbanUser":"9512bb945535dd9ef2fe1dacc60073ce01fc8d8f3e93fec6583ab2dc455d1309","AdminListNoteViews":"08631c2621fdb1e1265d238476428a5a108673be31c9fa3823233984adaee8ea","AdminListSubgraphs":"4e45ae80a24576cab70fdbb4790a0a7acbde1171823623ed8d7a00e495b596cd","AdminListUserBans":"69e1b4b4cb152647fa3474d44d1fccc7e5bfdc9bfa95d60b9726d4e52c94b3b4","AdminListUsers":"6f4fcb27423e59a080c8c0ba8cc8b69628bf9f961bdfbce5c62bf60c19db4075","AdminListUserSubgraphAccesses":"79ca5aafd82b91a579c3ea6232fa7a32773f2b74846785a00e0f0deee9854eca","AdminGraph":"39e949dd3c2f89603f0d09f5e348c2f46ee78f28d2412ff6efa9279ab68e457b","AdminUpdateNoteGraphPositions":"79055eb93ba30f15dc82d77bdce102ad7d30a32b24e57ebf3339507d9fc66fee","AdminSelectNoteView":"d158e0da61a2cbad548f8ada69595f5f5534f15275a06ef95b969d89a7edb3ad","AdminOffers":"b2fc36437eb7fc9ce54ed7fe52c5d6357eb56c1a2a739e52ed5a355499a817ce","AdminCreateOfferMutation":"54bfc14494b6645090140ee5379fef55991e303aa9bd6df0a022bb6b7285297f","AdminShowOffer":"644549579b7660594fa585b864956004f5a0ff8117877656f5089df760c1d79d","AdminUpdateOfferMutation":"bad9b0a25d99a5b3c45ca5116f543b11b7911d1c8e086ff532efd53f29908ca6","AdminPurchases":"47a76a00baca079d6e46244a8ba7bf75daf8063fdf530c5a8d46a876fdc00900","AdminMakeReleaseLive":"d0a1f7de2a4ea212ad0f7b80fd591943f26ca638c29e46acb604f47828db4204","AdminReleases":"a8954143d0ddc89b0a4d8ea0dfc63f3514e085d616d1089f8a0099adc9576993","AdminCreateRelease":"8fa0c84f61a4546c22ab6ccdafb6b18367eaae11e64dd35afeabdcaa9f19ee20","AdminBanUser":"b7eaca2436a0420749fd9818239ddaaf03cf39d13fb051af54e1ddfc16b0a376","AdminNoteView":"71287b914ef21187cc102872ffec82d7d3dbe8f1c7583391227f3829c627cc0d","AdminShowSubgraph":"e55ded2d39f46e61be846d0eaa24e9434a4cbc400e4d93fbde65dadeb14dd559","UpdateSubgraph":"a1ed76b00109cb6eac864a2c57d63d956d4c34676eb519e8d0a33abcd8c8945b","AdminUserSubgraphAccess":"a624b706534097050759fc343d7335c36f60b7fd8cb7eac35979d2c2e0b166da","AdminUpdateUserSubgraphAccess":"1f5c1a927c82b86a3d0e0313744fe51f023ed479f955e6846225bff4ab75a91d","AdminSelectSubgraphList":"bb432284d1aa05d873a33069ba3848e54368d1f8f98ee05fd948722f9a53f92d","AdminSelectSubgraph":"a801d4b303ea060e27e22011e2d7cc74c9a17a95e0877622e55e4012640c11a7","SignOut":"8e1a898d776a103a20b7dbdfbeb8196fce0175ffe38fa7af9da2848cefdadedf","RequestEmailSignInCode":"cc5a33407c1a3cca08dbcba6847a88298eec84b10f18449244cf13e458449ef9","SignInByEmail":"5678d2ca29b77529e0b9942845dcea2c941c6b5a234863dc86ac2ffaad668614","Viewer":"11c2c89ff045bd18461fa0409d26dcf4acbb063202aa8e74615fcd74d21db756","PaywallActivePurchaseQuery":"49eef6eeabb08d2251039c9b3ca67cade0c7af866c29300712400aa44d34d15b","PaywallQuery":"600ed11b093fa49a35a066487efd179edb8593e273565d02b13c87bb04cb49d7","CreatePaymentLink":"3c4ea8c746c573dc9a48a303c13f2b6f7ac3ccad23045f8e989105ea0df98b7a","UserSubscriptions":"1e67f2ab803afe77d6859c2130b2fede9f9edce8c9d841e14ce552c3ecf40115"}

export const $trip2g_graphql_payment_type = PaymentType;

export const $trip2g_graphql_role = Role;

}