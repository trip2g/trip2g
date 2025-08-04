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

export type ActiveOffers = {
  __typename?: 'ActiveOffers';
  nodes: Array<Offer>;
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

export type AdminBoostyCredentials = {
  __typename?: 'AdminBoostyCredentials';
  blogName: Scalars['String']['output'];
  createdAt: Scalars['Time']['output'];
  createdBy: AdminUser;
  deletedAt?: Maybe<Scalars['Time']['output']>;
  deletedBy?: Maybe<AdminUser>;
  deviceId: Scalars['String']['output'];
  id: Scalars['Int64']['output'];
  members: AdminBoostyMembersConnection;
  state: BoostyCredentialsStateEnum;
  tiers: AdminBoostyTiersConnection;
};

export type AdminBoostyCredentialsConnection = {
  __typename?: 'AdminBoostyCredentialsConnection';
  nodes: Array<AdminBoostyCredentials>;
};

export type AdminBoostyCredentialsFilterInput = {
  state?: InputMaybe<BoostyCredentialsStateEnum>;
};

export type AdminBoostyMember = {
  __typename?: 'AdminBoostyMember';
  boostyId: Scalars['Int64']['output'];
  createdAt: Scalars['Time']['output'];
  currentTier?: Maybe<AdminBoostyTier>;
  data: Scalars['String']['output'];
  email: Scalars['String']['output'];
  id: Scalars['Int64']['output'];
  missedAt?: Maybe<Scalars['Time']['output']>;
  status: Scalars['String']['output'];
};

export type AdminBoostyMembersConnection = {
  __typename?: 'AdminBoostyMembersConnection';
  nodes: Array<AdminBoostyMember>;
};

export type AdminBoostyTier = {
  __typename?: 'AdminBoostyTier';
  boostyId: Scalars['Int64']['output'];
  createdAt: Scalars['Time']['output'];
  data: Scalars['String']['output'];
  id: Scalars['Int64']['output'];
  missedAt?: Maybe<Scalars['Time']['output']>;
  name: Scalars['String']['output'];
  subgraphs: Array<AdminSubgraph>;
};

export type AdminBoostyTiersConnection = {
  __typename?: 'AdminBoostyTiersConnection';
  nodes: Array<AdminBoostyTier>;
};

export type AdminLatestNoteViewsConnection = {
  __typename?: 'AdminLatestNoteViewsConnection';
  nodes: Array<NoteView>;
};

export type AdminLatestNoteViewsFilter = {
  withWarnings?: InputMaybe<Scalars['Boolean']['input']>;
};

export type AdminMutation = {
  __typename?: 'AdminMutation';
  banUser: BanUserOrErrorPayload;
  createApiKey: CreateApiKeyOrErrorPayload;
  createBoostyCredentials: CreateBoostyCredentialsOrErrorPayload;
  createNotFoundIgnoredPattern: CreateNotFoundIgnoredPatternOrErrorPayload;
  createOffer: CreateOfferOrErrorPayload;
  createPatreonCredentials: CreatePatreonCredentialsOrErrorPayload;
  createRedirect: CreateRedirectOrErrorPayload;
  createRelease: CreateReleaseOrErrorPayload;
  createTgBot: CreateTgBotOrErrorPayload;
  deleteBoostyCredentials: DeleteBoostyCredentialsOrErrorPayload;
  deleteNotFoundIgnoredPattern: DeleteNotFoundIgnoredPatternOrErrorPayload;
  deletePatreonCredentials: DeletePatreonCredentialsOrErrorPayload;
  deleteRedirect: DeleteRedirectOrErrorPayload;
  disableApiKey: DisableApiKeyOrErrorPayload;
  makeReleaseLive: MakeReleaseLiveOrErrorPayload;
  refreshBoostyData: RefreshBoostyDataOrErrorPayload;
  refreshPatreonData: RefreshPatreonDataOrErrorPayload;
  resetNotFoundPath: ResetNotFoundPathOrErrorPayload;
  restoreBoostyCredentials: RestoreBoostyCredentialsOrErrorPayload;
  restorePatreonCredentials: RestorePatreonCredentialsOrErrorPayload;
  setBoostyTierSubgraphs: SetBoostyTierSubgraphsOrErrorPayload;
  setPatreonTierSubgraphs: SetPatreonTierSubgraphsOrErrorPayload;
  setTgChatSubgraphs: SetTgChatSubgraphsOrErrorPayload;
  unbanUser: UnbanUserOrErrorPayload;
  updateBoostyCredentials: UpdateBoostyCredentialsOrErrorPayload;
  updateNotFoundIgnoredPattern: UpdateNotFoundIgnoredPatternOrErrorPayload;
  updateNoteGraphPositions: UpdateNoteGraphPositionsOrErrorPayload;
  updateOffer: UpdateOfferOrErrorPayload;
  updateRedirect: UpdateRedirectOrErrorPayload;
  updateSubgraph: UpdateSubgraphOrErrorPayload;
  updateTgBot: UpdateTgBotOrErrorPayload;
  updateUserSubgraphAccess: UpdateUserSubgraphAccessOrErrorPayload;
};


export type AdminMutationBanUserArgs = {
  input: BanUserInput;
};


export type AdminMutationCreateApiKeyArgs = {
  input: CreateApiKeyInput;
};


export type AdminMutationCreateBoostyCredentialsArgs = {
  input: CreateBoostyCredentialsInput;
};


export type AdminMutationCreateNotFoundIgnoredPatternArgs = {
  input: CreateNotFoundIgnoredPatternInput;
};


export type AdminMutationCreateOfferArgs = {
  input: CreateOfferInput;
};


export type AdminMutationCreatePatreonCredentialsArgs = {
  input: CreatePatreonCredentialsInput;
};


export type AdminMutationCreateRedirectArgs = {
  input: CreateRedirectInput;
};


export type AdminMutationCreateReleaseArgs = {
  input: CreateReleaseInput;
};


export type AdminMutationCreateTgBotArgs = {
  input: CreateTgBotInput;
};


export type AdminMutationDeleteBoostyCredentialsArgs = {
  input: DeleteBoostyCredentialsInput;
};


export type AdminMutationDeleteNotFoundIgnoredPatternArgs = {
  input: DeleteNotFoundIgnoredPatternInput;
};


export type AdminMutationDeletePatreonCredentialsArgs = {
  input: DeletePatreonCredentialsInput;
};


export type AdminMutationDeleteRedirectArgs = {
  input: DeleteRedirectInput;
};


export type AdminMutationDisableApiKeyArgs = {
  input: DisableApiKeyInput;
};


export type AdminMutationMakeReleaseLiveArgs = {
  input: MakeReleaseLiveInput;
};


export type AdminMutationRefreshBoostyDataArgs = {
  input: RefreshBoostyDataInput;
};


export type AdminMutationRefreshPatreonDataArgs = {
  input: RefreshPatreonDataInput;
};


export type AdminMutationResetNotFoundPathArgs = {
  input: ResetNotFoundPathInput;
};


export type AdminMutationRestoreBoostyCredentialsArgs = {
  input: RestoreBoostyCredentialsInput;
};


export type AdminMutationRestorePatreonCredentialsArgs = {
  input: RestorePatreonCredentialsInput;
};


export type AdminMutationSetBoostyTierSubgraphsArgs = {
  input: SetBoostyTierSubgraphsInput;
};


export type AdminMutationSetPatreonTierSubgraphsArgs = {
  input: SetPatreonTierSubgraphsInput;
};


export type AdminMutationSetTgChatSubgraphsArgs = {
  input: SetTgChatSubgraphsInput;
};


export type AdminMutationUnbanUserArgs = {
  input: UnbanUserInput;
};


export type AdminMutationUpdateBoostyCredentialsArgs = {
  input: UpdateBoostyCredentialsInput;
};


export type AdminMutationUpdateNotFoundIgnoredPatternArgs = {
  input: UpdateNotFoundIgnoredPatternInput;
};


export type AdminMutationUpdateNoteGraphPositionsArgs = {
  input: UpdateNoteGraphPositionsInput;
};


export type AdminMutationUpdateOfferArgs = {
  input: UpdateOfferInput;
};


export type AdminMutationUpdateRedirectArgs = {
  input: UpdateRedirectInput;
};


export type AdminMutationUpdateSubgraphArgs = {
  input: UpdateSubgraphInput;
};


export type AdminMutationUpdateTgBotArgs = {
  input: UpdateTgBotInput;
};


export type AdminMutationUpdateUserSubgraphAccessArgs = {
  input: UpdateUserSubgraphAccessInput;
};

export type AdminNotFoundIgnoredPattern = {
  __typename?: 'AdminNotFoundIgnoredPattern';
  createdAt: Scalars['Time']['output'];
  createdBy: AdminUser;
  id: Scalars['Int64']['output'];
  pattern: Scalars['String']['output'];
};

export type AdminNotFoundIgnoredPatternsConnection = {
  __typename?: 'AdminNotFoundIgnoredPatternsConnection';
  nodes: Array<AdminNotFoundIgnoredPattern>;
};

export type AdminNotFoundPath = {
  __typename?: 'AdminNotFoundPath';
  id: Scalars['Int64']['output'];
  lastHitAt: Scalars['Time']['output'];
  path: Scalars['String']['output'];
  totalHits: Scalars['Int64']['output'];
};

export type AdminNotFoundPathsConnection = {
  __typename?: 'AdminNotFoundPathsConnection';
  nodes: Array<AdminNotFoundPath>;
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

export type AdminPatreonCampaign = {
  __typename?: 'AdminPatreonCampaign';
  attributes: Scalars['String']['output'];
  campaignID: Scalars['String']['output'];
  createdAt: Scalars['Time']['output'];
  credentialsID: Scalars['Int64']['output'];
  id: Scalars['Int64']['output'];
  missedAt?: Maybe<Scalars['Time']['output']>;
};

export type AdminPatreonCredentials = {
  __typename?: 'AdminPatreonCredentials';
  createdAt: Scalars['Time']['output'];
  createdBy: AdminUser;
  creatorAccessToken: Scalars['String']['output'];
  deletedAt?: Maybe<Scalars['Time']['output']>;
  deletedBy?: Maybe<AdminUser>;
  id: Scalars['Int64']['output'];
  members: AdminPatreonMembersConnection;
  state: PatreonCredentialsStateEnum;
  syncedAt?: Maybe<Scalars['Time']['output']>;
  tiers: AdminPatreonTiersConnection;
};

export type AdminPatreonCredentialsConnection = {
  __typename?: 'AdminPatreonCredentialsConnection';
  nodes: Array<AdminPatreonCredentials>;
};

export type AdminPatreonCredentialsFilterInput = {
  state?: InputMaybe<PatreonCredentialsStateEnum>;
};

export type AdminPatreonMember = {
  __typename?: 'AdminPatreonMember';
  campaignID: Scalars['Int64']['output'];
  currentTier?: Maybe<AdminPatreonTier>;
  currentTierID?: Maybe<Scalars['Int64']['output']>;
  email: Scalars['String']['output'];
  id: Scalars['Int64']['output'];
  patreonID: Scalars['String']['output'];
  status: Scalars['String']['output'];
};

export type AdminPatreonMembersConnection = {
  __typename?: 'AdminPatreonMembersConnection';
  nodes: Array<AdminPatreonMember>;
};

export type AdminPatreonTier = {
  __typename?: 'AdminPatreonTier';
  amountCents: Scalars['Int64']['output'];
  attributes: Scalars['String']['output'];
  campaignID: Scalars['Int64']['output'];
  createdAt: Scalars['Time']['output'];
  id: Scalars['Int64']['output'];
  missedAt?: Maybe<Scalars['Time']['output']>;
  subgraphs: Array<AdminSubgraph>;
  tierID: Scalars['String']['output'];
  title: Scalars['String']['output'];
};

export type AdminPatreonTiersConnection = {
  __typename?: 'AdminPatreonTiersConnection';
  nodes: Array<AdminPatreonTier>;
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
  allBoostyCredentials: AdminBoostyCredentialsConnection;
  allLatestNoteViews: AdminLatestNoteViewsConnection;
  allNotFoundIgnoredPatterns: AdminNotFoundIgnoredPatternsConnection;
  allNotFoundPaths: AdminNotFoundPathsConnection;
  allOffers: AdminOffersConnection;
  allPatreonCredentials: AdminPatreonCredentialsConnection;
  allPurchases: AdminPurchasesConnection;
  allRedirects: AdminRedirectsConnection;
  allReleases: AdminReleasesConnection;
  allSubgraphs: AdminSubgraphsConnection;
  allTgBots: AdminTgBotsConnection;
  allUserSubgraphAccesses: AdminUserSubgraphAccessesConnection;
  allUserUserBans: AdminUserBansConnection;
  allUsers: AdminUsersConnection;
  apiKeyLogs: AdminApiKeyLogsConnection;
  boostyCredentials?: Maybe<AdminBoostyCredentials>;
  noteView?: Maybe<NoteView>;
  offer?: Maybe<AdminOffer>;
  patreonCredentials?: Maybe<AdminPatreonCredentials>;
  purchase?: Maybe<AdminPurchase>;
  redirect?: Maybe<AdminRedirect>;
  subgraph?: Maybe<AdminSubgraph>;
  tgBot?: Maybe<AdminTgBot>;
  tgBotChats: AdminTgBotChatsConnection;
  tgChatMembers: AdminTgChatMembersConnection;
  tgChatSubgraphAccesses: AdminTgChatSubgraphAccessesConnection;
  userSubgraphAccess?: Maybe<AdminUserSubgraphAccess>;
};


export type AdminQueryAllBoostyCredentialsArgs = {
  filter?: InputMaybe<AdminBoostyCredentialsFilterInput>;
};


export type AdminQueryAllLatestNoteViewsArgs = {
  filter?: InputMaybe<AdminLatestNoteViewsFilter>;
};


export type AdminQueryAllPatreonCredentialsArgs = {
  filter?: InputMaybe<AdminPatreonCredentialsFilterInput>;
};


export type AdminQueryApiKeyLogsArgs = {
  filter: ApiKeyLogsFilterInput;
};


export type AdminQueryBoostyCredentialsArgs = {
  id: Scalars['Int64']['input'];
};


export type AdminQueryNoteViewArgs = {
  id: Scalars['String']['input'];
};


export type AdminQueryOfferArgs = {
  id: Scalars['Int64']['input'];
};


export type AdminQueryPatreonCredentialsArgs = {
  id: Scalars['Int64']['input'];
};


export type AdminQueryPurchaseArgs = {
  id: Scalars['String']['input'];
};


export type AdminQueryRedirectArgs = {
  id: Scalars['Int64']['input'];
};


export type AdminQuerySubgraphArgs = {
  id: Scalars['Int64']['input'];
};


export type AdminQueryTgBotArgs = {
  id: Scalars['Int64']['input'];
};


export type AdminQueryTgBotChatsArgs = {
  filter: AdminTgBotChatsFilterInput;
};


export type AdminQueryTgChatMembersArgs = {
  filter: AdminTgChatMembersFilterInput;
};


export type AdminQueryTgChatSubgraphAccessesArgs = {
  filter: AdminTgChatSubgraphAccessesFilterInput;
};


export type AdminQueryUserSubgraphAccessArgs = {
  id: Scalars['Int64']['input'];
};

export type AdminRedirect = {
  __typename?: 'AdminRedirect';
  createdAt: Scalars['Time']['output'];
  createdBy: AdminUser;
  id: Scalars['Int64']['output'];
  ignoreCase: Scalars['Boolean']['output'];
  isRegex: Scalars['Boolean']['output'];
  pattern: Scalars['String']['output'];
  target: Scalars['String']['output'];
};

export type AdminRedirectsConnection = {
  __typename?: 'AdminRedirectsConnection';
  nodes: Array<AdminRedirect>;
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
  hidden: Scalars['Boolean']['output'];
  id: Scalars['Int64']['output'];
  name: Scalars['String']['output'];
};

export type AdminSubgraphsConnection = {
  __typename?: 'AdminSubgraphsConnection';
  nodes: Array<AdminSubgraph>;
};

export type AdminTgBot = {
  __typename?: 'AdminTgBot';
  createdAt: Scalars['Time']['output'];
  createdBy: AdminUser;
  description: Scalars['String']['output'];
  enabled: Scalars['Boolean']['output'];
  id: Scalars['Int64']['output'];
  name: Scalars['String']['output'];
};

export type AdminTgBotChat = {
  __typename?: 'AdminTgBotChat';
  addedAt: Scalars['Time']['output'];
  chatTitle: Scalars['String']['output'];
  chatType: Scalars['String']['output'];
  id: Scalars['Int64']['output'];
  memberCount: Scalars['Int']['output'];
  removedAt?: Maybe<Scalars['Time']['output']>;
  subgraphAccesses: Array<AdminTgChatSubgraphAccess>;
};

export type AdminTgBotChatsConnection = {
  __typename?: 'AdminTgBotChatsConnection';
  nodes: Array<AdminTgBotChat>;
};

export type AdminTgBotChatsFilterInput = {
  botId?: InputMaybe<Scalars['Int64']['input']>;
  includeRemoved?: InputMaybe<Scalars['Boolean']['input']>;
};

export type AdminTgBotsConnection = {
  __typename?: 'AdminTgBotsConnection';
  nodes: Array<AdminTgBot>;
};

export type AdminTgChatMember = {
  __typename?: 'AdminTgChatMember';
  chatId: Scalars['Int64']['output'];
  createdAt: Scalars['Time']['output'];
  profile?: Maybe<AdminTgUserProfile>;
  userId: Scalars['Int64']['output'];
};

export type AdminTgChatMembersConnection = {
  __typename?: 'AdminTgChatMembersConnection';
  nodes: Array<AdminTgChatMember>;
};

export type AdminTgChatMembersFilterInput = {
  chatId: Scalars['Int64']['input'];
};

export type AdminTgChatSubgraphAccess = {
  __typename?: 'AdminTgChatSubgraphAccess';
  chat: AdminTgBotChat;
  chatId: Scalars['Int64']['output'];
  createdAt: Scalars['Time']['output'];
  id: Scalars['Int64']['output'];
  subgraph: AdminSubgraph;
  subgraphId: Scalars['Int64']['output'];
};

export type AdminTgChatSubgraphAccessesConnection = {
  __typename?: 'AdminTgChatSubgraphAccessesConnection';
  nodes: Array<AdminTgChatSubgraphAccess>;
};

export type AdminTgChatSubgraphAccessesFilterInput = {
  chatId?: InputMaybe<Scalars['Int64']['input']>;
  subgraphId?: InputMaybe<Scalars['Int64']['input']>;
};

export type AdminTgUserProfile = {
  __typename?: 'AdminTgUserProfile';
  botId: Scalars['Int64']['output'];
  chatId: Scalars['Int64']['output'];
  createdAt: Scalars['Time']['output'];
  firstName?: Maybe<Scalars['String']['output']>;
  lastName?: Maybe<Scalars['String']['output']>;
  sha256Hash: Scalars['String']['output'];
  username?: Maybe<Scalars['String']['output']>;
};

export type AdminUser = {
  __typename?: 'AdminUser';
  ban?: Maybe<UserBan>;
  createdAt: Scalars['Time']['output'];
  email?: Maybe<Scalars['String']['output']>;
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

export enum BoostyCredentialsStateEnum {
  Active = 'ACTIVE',
  Deleted = 'DELETED'
}

export type CreateApiKeyInput = {
  description: Scalars['String']['input'];
};

export type CreateApiKeyOrErrorPayload = CreateApiKeyPayload | ErrorPayload;

export type CreateApiKeyPayload = {
  __typename?: 'CreateApiKeyPayload';
  apiKey: AdminApiKey;
  value: Scalars['String']['output'];
};

export type CreateBoostyCredentialsInput = {
  authData: Scalars['String']['input'];
  blogName: Scalars['String']['input'];
  deviceId: Scalars['String']['input'];
};

export type CreateBoostyCredentialsOrErrorPayload = CreateBoostyCredentialsPayload | ErrorPayload;

export type CreateBoostyCredentialsPayload = {
  __typename?: 'CreateBoostyCredentialsPayload';
  boostyCredentials: AdminBoostyCredentials;
};

export type CreateEmailWaitListRequestInput = {
  email: Scalars['String']['input'];
  pathId: Scalars['Int64']['input'];
};

export type CreateEmailWaitListRequestOrErrorPayload = CreateEmailWaitListRequestPayload | ErrorPayload;

export type CreateEmailWaitListRequestPayload = {
  __typename?: 'CreateEmailWaitListRequestPayload';
  success: Scalars['Boolean']['output'];
};

export type CreateNotFoundIgnoredPatternInput = {
  pattern: Scalars['String']['input'];
};

export type CreateNotFoundIgnoredPatternOrErrorPayload = CreateNotFoundIgnoredPatternPayload | ErrorPayload;

export type CreateNotFoundIgnoredPatternPayload = {
  __typename?: 'CreateNotFoundIgnoredPatternPayload';
  notFoundIgnoredPattern: AdminNotFoundIgnoredPattern;
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

export type CreatePatreonCredentialsInput = {
  creatorAccessToken: Scalars['String']['input'];
};

export type CreatePatreonCredentialsOrErrorPayload = CreatePatreonCredentialsPayload | ErrorPayload;

export type CreatePatreonCredentialsPayload = {
  __typename?: 'CreatePatreonCredentialsPayload';
  patreonCredentials: AdminPatreonCredentials;
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

export type CreateRedirectInput = {
  ignoreCase: Scalars['Boolean']['input'];
  isRegex: Scalars['Boolean']['input'];
  pattern: Scalars['String']['input'];
  target: Scalars['String']['input'];
};

export type CreateRedirectOrErrorPayload = CreateRedirectPayload | ErrorPayload;

export type CreateRedirectPayload = {
  __typename?: 'CreateRedirectPayload';
  redirect: AdminRedirect;
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

export type CreateTgBotInput = {
  description: Scalars['String']['input'];
  token: Scalars['String']['input'];
};

export type CreateTgBotOrErrorPayload = CreateTgBotPayload | ErrorPayload;

export type CreateTgBotPayload = {
  __typename?: 'CreateTgBotPayload';
  tgBot: AdminTgBot;
};

export type DeleteBoostyCredentialsInput = {
  id: Scalars['Int64']['input'];
};

export type DeleteBoostyCredentialsOrErrorPayload = DeleteBoostyCredentialsPayload | ErrorPayload;

export type DeleteBoostyCredentialsPayload = {
  __typename?: 'DeleteBoostyCredentialsPayload';
  boostyCredentials: AdminBoostyCredentials;
  deletedId: Scalars['Int64']['output'];
};

export type DeleteNotFoundIgnoredPatternInput = {
  id: Scalars['Int64']['input'];
};

export type DeleteNotFoundIgnoredPatternOrErrorPayload = DeleteNotFoundIgnoredPatternPayload | ErrorPayload;

export type DeleteNotFoundIgnoredPatternPayload = {
  __typename?: 'DeleteNotFoundIgnoredPatternPayload';
  deletedId: Scalars['Int64']['output'];
};

export type DeletePatreonCredentialsInput = {
  id: Scalars['Int64']['input'];
};

export type DeletePatreonCredentialsOrErrorPayload = DeletePatreonCredentialsPayload | ErrorPayload;

export type DeletePatreonCredentialsPayload = {
  __typename?: 'DeletePatreonCredentialsPayload';
  deletedId: Scalars['Int64']['output'];
  patreonCredentials: AdminPatreonCredentials;
};

export type DeleteRedirectInput = {
  id: Scalars['Int64']['input'];
};

export type DeleteRedirectOrErrorPayload = DeleteRedirectPayload | ErrorPayload;

export type DeleteRedirectPayload = {
  __typename?: 'DeleteRedirectPayload';
  id: Scalars['Int64']['output'];
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

export type HideNotesInput = {
  paths: Array<Scalars['String']['input']>;
};

export type HideNotesOrErrorPayload = ErrorPayload | HideNotesPayload;

export type HideNotesPayload = {
  __typename?: 'HideNotesPayload';
  success: Scalars['Boolean']['output'];
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
  createEmailWaitListRequest: CreateEmailWaitListRequestOrErrorPayload;
  createPaymentLink: CreatePaymentLinkOrErrorPayload;
  hideNotes: HideNotesOrErrorPayload;
  pushNotes: PushNotesOrErrorPayload;
  requestEmailSignInCode: RequestEmailSignInCodeOrErrorPayload;
  signInByEmail: SignInOrErrorPayload;
  signOut: SignOutOrErrorPayload;
  uploadNoteAsset: UploadNoteAssetOrErrorPayload;
};


export type MutationCreateEmailWaitListRequestArgs = {
  input: CreateEmailWaitListRequestInput;
};


export type MutationCreatePaymentLinkArgs = {
  input: CreatePaymentLinkInput;
};


export type MutationHideNotesArgs = {
  input: HideNotesInput;
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

export type NoteInput = {
  path: Scalars['String']['input'];
  referer: Scalars['String']['input'];
};

export type NotePath = {
  __typename?: 'NotePath';
  latestContentHash: Scalars['String']['output'];
  value: Scalars['String']['output'];
};

export type NoteTocItem = {
  __typename?: 'NoteTocItem';
  id: Scalars['String']['output'];
  level: Scalars['Int']['output'];
  title: Scalars['String']['output'];
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
  warnings: Array<NoteWarning>;
};

export type NoteWarning = {
  __typename?: 'NoteWarning';
  level: NoteWarningLevelEnum;
  message: Scalars['String']['output'];
};

export enum NoteWarningLevelEnum {
  Critical = 'CRITICAL',
  Info = 'INFO',
  Warning = 'WARNING'
}

export type Offer = {
  __typename?: 'Offer';
  id: Scalars['String']['output'];
  priceUSD: Scalars['Float']['output'];
  subgraphs: Array<Subgraph>;
};

export enum PatreonCredentialsStateEnum {
  Active = 'ACTIVE',
  Deleted = 'DELETED'
}

export enum PaymentType {
  Crypto = 'CRYPTO'
}

export type PublicNote = {
  __typename?: 'PublicNote';
  html: Scalars['String']['output'];
  title: Scalars['String']['output'];
  toc: Array<NoteTocItem>;
};

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
  note?: Maybe<PublicNote>;
  notePaths: Array<NotePath>;
  viewer: Viewer;
};


export type QueryNoteArgs = {
  input: NoteInput;
};

export type RefreshBoostyDataInput = {
  credentialsId: Scalars['Int64']['input'];
};

export type RefreshBoostyDataOrErrorPayload = ErrorPayload | RefreshBoostyDataPayload;

export type RefreshBoostyDataPayload = {
  __typename?: 'RefreshBoostyDataPayload';
  credentials: AdminBoostyCredentials;
  credentialsID: Scalars['Int64']['output'];
  success: Scalars['Boolean']['output'];
};

export type RefreshPatreonDataInput = {
  credentialsId: Scalars['Int64']['input'];
};

export type RefreshPatreonDataOrErrorPayload = ErrorPayload | RefreshPatreonDataPayload;

export type RefreshPatreonDataPayload = {
  __typename?: 'RefreshPatreonDataPayload';
  credentials: AdminPatreonCredentials;
  credentialsID: Scalars['Int64']['output'];
  success: Scalars['Boolean']['output'];
};

export type RequestEmailSignInCodeInput = {
  email: Scalars['String']['input'];
};

export type RequestEmailSignInCodeOrErrorPayload = ErrorPayload | RequestEmailSignInCodePayload;

export type RequestEmailSignInCodePayload = {
  __typename?: 'RequestEmailSignInCodePayload';
  success: Scalars['Boolean']['output'];
};

export type ResetNotFoundPathInput = {
  id: Scalars['Int64']['input'];
};

export type ResetNotFoundPathOrErrorPayload = ErrorPayload | ResetNotFoundPathPayload;

export type ResetNotFoundPathPayload = {
  __typename?: 'ResetNotFoundPathPayload';
  notFoundPath: AdminNotFoundPath;
};

export type RestoreBoostyCredentialsInput = {
  id: Scalars['Int64']['input'];
};

export type RestoreBoostyCredentialsOrErrorPayload = ErrorPayload | RestoreBoostyCredentialsPayload;

export type RestoreBoostyCredentialsPayload = {
  __typename?: 'RestoreBoostyCredentialsPayload';
  boostyCredentials: AdminBoostyCredentials;
};

export type RestorePatreonCredentialsInput = {
  id: Scalars['Int64']['input'];
};

export type RestorePatreonCredentialsOrErrorPayload = ErrorPayload | RestorePatreonCredentialsPayload;

export type RestorePatreonCredentialsPayload = {
  __typename?: 'RestorePatreonCredentialsPayload';
  patreonCredentials: AdminPatreonCredentials;
};

export enum Role {
  Admin = 'ADMIN',
  Guest = 'GUEST',
  User = 'USER'
}

export type SetBoostyTierSubgraphsInput = {
  subgraphIds: Array<Scalars['Int64']['input']>;
  tierId: Scalars['Int64']['input'];
};

export type SetBoostyTierSubgraphsOrErrorPayload = ErrorPayload | SetBoostyTierSubgraphsPayload;

export type SetBoostyTierSubgraphsPayload = {
  __typename?: 'SetBoostyTierSubgraphsPayload';
  success: Scalars['Boolean']['output'];
  tier: AdminBoostyTier;
};

export type SetPatreonTierSubgraphsInput = {
  subgraphIds: Array<Scalars['Int64']['input']>;
  tierId: Scalars['Int64']['input'];
};

export type SetPatreonTierSubgraphsOrErrorPayload = ErrorPayload | SetPatreonTierSubgraphsPayload;

export type SetPatreonTierSubgraphsPayload = {
  __typename?: 'SetPatreonTierSubgraphsPayload';
  success: Scalars['Boolean']['output'];
  tier: AdminPatreonTier;
};

export type SetTgChatSubgraphsInput = {
  chatId: Scalars['Int64']['input'];
  subgraphIds: Array<Scalars['Int64']['input']>;
};

export type SetTgChatSubgraphsOrErrorPayload = ErrorPayload | SetTgChatSubgraphsPayload;

export type SetTgChatSubgraphsPayload = {
  __typename?: 'SetTgChatSubgraphsPayload';
  chat: AdminTgBotChat;
  success: Scalars['Boolean']['output'];
};

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

export type SubgraphWaitList = {
  __typename?: 'SubgraphWaitList';
  emailAllowed: Scalars['Boolean']['output'];
  tgBotUrl?: Maybe<Scalars['String']['output']>;
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

export type UpdateBoostyCredentialsInput = {
  authData?: InputMaybe<Scalars['String']['input']>;
  blogName?: InputMaybe<Scalars['String']['input']>;
  deviceId?: InputMaybe<Scalars['String']['input']>;
  id: Scalars['Int64']['input'];
};

export type UpdateBoostyCredentialsOrErrorPayload = ErrorPayload | UpdateBoostyCredentialsPayload;

export type UpdateBoostyCredentialsPayload = {
  __typename?: 'UpdateBoostyCredentialsPayload';
  boostyCredentials: AdminBoostyCredentials;
};

export type UpdateNotFoundIgnoredPatternInput = {
  id: Scalars['Int64']['input'];
  pattern: Scalars['String']['input'];
};

export type UpdateNotFoundIgnoredPatternOrErrorPayload = ErrorPayload | UpdateNotFoundIgnoredPatternPayload;

export type UpdateNotFoundIgnoredPatternPayload = {
  __typename?: 'UpdateNotFoundIgnoredPatternPayload';
  notFoundIgnoredPattern: AdminNotFoundIgnoredPattern;
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

export type UpdateRedirectInput = {
  id: Scalars['Int64']['input'];
  ignoreCase: Scalars['Boolean']['input'];
  isRegex: Scalars['Boolean']['input'];
  pattern: Scalars['String']['input'];
  target: Scalars['String']['input'];
};

export type UpdateRedirectOrErrorPayload = ErrorPayload | UpdateRedirectPayload;

export type UpdateRedirectPayload = {
  __typename?: 'UpdateRedirectPayload';
  redirect: AdminRedirect;
};

export type UpdateSubgraphInput = {
  color: Scalars['String']['input'];
  hidden: Scalars['Boolean']['input'];
  id: Scalars['Int64']['input'];
};

export type UpdateSubgraphOrErrorPayload = ErrorPayload | UpdateSubgraphPayload;

export type UpdateSubgraphPayload = {
  __typename?: 'UpdateSubgraphPayload';
  subgraph: AdminSubgraph;
};

export type UpdateTgBotInput = {
  description?: InputMaybe<Scalars['String']['input']>;
  enabled?: InputMaybe<Scalars['Boolean']['input']>;
  id: Scalars['Int64']['input'];
};

export type UpdateTgBotOrErrorPayload = ErrorPayload | UpdateTgBotPayload;

export type UpdateTgBotPayload = {
  __typename?: 'UpdateTgBotPayload';
  tgBot: AdminTgBot;
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
  email?: Maybe<Scalars['String']['output']>;
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
  offers?: Maybe<ViewerOffers>;
  role: Role;
  user?: Maybe<User>;
};


export type ViewerOffersArgs = {
  filter: ViewerOffersFilter;
};

export type ViewerOffers = ActiveOffers | SubgraphWaitList;

export type ViewerOffersFilter = {
  pageId?: InputMaybe<Scalars['Int64']['input']>;
};

export type AdminsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allAdmins: { __typename?: 'AdminAdminsConnection', nodes: Array<{ __typename?: 'Admin', id: any, grantedAt: any, user: { __typename?: 'AdminUser', email?: string | null } }> } } };

export type DisableApiKeyMutationVariables = Exact<{
  input: DisableApiKeyInput;
}>;


export type DisableApiKeyMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'DisableApiKeyPayload', apiKey: { __typename?: 'AdminApiKey', id: any } } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminListApiKeysQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminListApiKeysQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allApiKeys: { __typename?: 'AdminApiKeysConnection', nodes: Array<{ __typename?: 'AdminApiKey', id: any, createdAt: any, description: string, disabledAt?: any | null, createdBy: { __typename?: 'AdminUser', id: any, email?: string | null }, disabledBy?: { __typename?: 'AdminUser', id: any, email?: string | null } | null }> } } };

export type AdminCreateApiKeyMutationVariables = Exact<{
  input: CreateApiKeyInput;
}>;


export type AdminCreateApiKeyMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'CreateApiKeyPayload', value: string, apiKey: { __typename?: 'AdminApiKey', id: any } } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminApiKeyShowQueryQueryVariables = Exact<{
  filter: ApiKeyLogsFilterInput;
}>;


export type AdminApiKeyShowQueryQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', apiKeyLogs: { __typename?: 'AdminApiKeyLogsConnection', nodes: Array<{ __typename?: 'AdminApiKeyLog', createdAt: any, actionName: string, ip: string }> } } };

export type AdminDeleteBoostyCredentialsMutationVariables = Exact<{
  input: DeleteBoostyCredentialsInput;
}>;


export type AdminDeleteBoostyCredentialsMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', deleteBoostyCredentials: { __typename?: 'DeleteBoostyCredentialsPayload', boostyCredentials: { __typename?: 'AdminBoostyCredentials', id: any } } | { __typename?: 'ErrorPayload', message: string } } };

export type RefreshBoostyDataMutationVariables = Exact<{
  input: RefreshBoostyDataInput;
}>;


export type RefreshBoostyDataMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', refreshBoostyData: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'RefreshBoostyDataPayload', success: boolean, credentials: { __typename?: 'AdminBoostyCredentials', id: any } } } };

export type AdminRestoreBoostyCredentialsMutationVariables = Exact<{
  input: RestoreBoostyCredentialsInput;
}>;


export type AdminRestoreBoostyCredentialsMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', restoreBoostyCredentials: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'RestoreBoostyCredentialsPayload', boostyCredentials: { __typename?: 'AdminBoostyCredentials', id: any } } } };

export type AdminBoostyCredentialsQueryVariables = Exact<{
  filter?: InputMaybe<AdminBoostyCredentialsFilterInput>;
}>;


export type AdminBoostyCredentialsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allBoostyCredentials: { __typename?: 'AdminBoostyCredentialsConnection', nodes: Array<{ __typename?: 'AdminBoostyCredentials', id: any, state: BoostyCredentialsStateEnum, deviceId: string, blogName: string, createdAt: any, createdBy: { __typename?: 'AdminUser', id: any, email?: string | null } }> } } };

export type AdminCreateBoostyCredsMutationVariables = Exact<{
  input: CreateBoostyCredentialsInput;
}>;


export type AdminCreateBoostyCredsMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', createBoostyCredentials: { __typename?: 'CreateBoostyCredentialsPayload', boostyCredentials: { __typename?: 'AdminBoostyCredentials', id: any } } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminBoostyCredentialsByIdQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminBoostyCredentialsByIdQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', boostyCredentials?: { __typename?: 'AdminBoostyCredentials', createdAt: any, deviceId: string, blogName: string, state: BoostyCredentialsStateEnum, createdBy: { __typename?: 'AdminUser', email?: string | null }, tiers: { __typename?: 'AdminBoostyTiersConnection', nodes: Array<{ __typename?: 'AdminBoostyTier', id: any, name: string, subgraphs: Array<{ __typename?: 'AdminSubgraph', id: any }> }> }, members: { __typename?: 'AdminBoostyMembersConnection', nodes: Array<{ __typename?: 'AdminBoostyMember', email: string, status: string }> } } | null } };

export type AdminBoostycredentialsShowSubgraphsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminBoostycredentialsShowSubgraphsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename?: 'AdminSubgraph', id: any, name: string }> } } };

export type AdminBoostycredentialsShowSubgraphsSaveMutationVariables = Exact<{
  input: SetBoostyTierSubgraphsInput;
}>;


export type AdminBoostycredentialsShowSubgraphsSaveMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'SetBoostyTierSubgraphsPayload', success: boolean } } };

export type AdminUnbanUserMutationVariables = Exact<{
  input: UnbanUserInput;
}>;


export type AdminUnbanUserMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'UnbanUserPayload', user: { __typename: 'AdminUser', id: any } } } };

export type AdminListNoteViewsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminListNoteViewsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allLatestNoteViews: { __typename?: 'AdminLatestNoteViewsConnection', nodes: Array<{ __typename?: 'NoteView', id: string, path: string, title: string, free: boolean }> } } };

export type AdminListSubgraphsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminListSubgraphsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename: 'AdminSubgraph', id: any, name: string, color?: string | null, createdAt: any }> } } };

export type AdminListUserBansQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminListUserBansQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allUserUserBans: { __typename?: 'AdminUserBansConnection', nodes: Array<{ __typename?: 'UserBan', createdAt: any, reason: string, id: any, user: { __typename: 'AdminUser', email?: string | null }, bannedBy?: { __typename?: 'Admin', user: { __typename?: 'AdminUser', email?: string | null } } | null }> } } };

export type AdminListUsersQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminListUsersQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allUsers: { __typename?: 'AdminUsersConnection', nodes: Array<{ __typename?: 'AdminUser', id: any, email?: string | null, createdAt: any, ban?: { __typename?: 'UserBan', reason: string } | null }> } } };

export type AdminListUserSubgraphAccessesQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminListUserSubgraphAccessesQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', data: { __typename?: 'AdminUserSubgraphAccessesConnection', nodes: Array<{ __typename: 'AdminUserSubgraphAccess', id: any, createdAt: any, expiresAt?: any | null, subgraph: { __typename?: 'AdminSubgraph', name: string }, user: { __typename?: 'AdminUser', id: any, email?: string | null } }> } } };

export type AdminGraphQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminGraphQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename?: 'AdminSubgraph', name: string, color?: string | null }> }, allLatestNoteViews: { __typename?: 'AdminLatestNoteViewsConnection', nodes: Array<{ __typename?: 'NoteView', id: string, subgraphNames: Array<string>, pathId: any, free: boolean, isHomePage: boolean, graphPosition?: { __typename?: 'Vector2', x: number, y: number } | null, inLinks: Array<{ __typename?: 'NoteView', title: string, pathId: any, id: string }> }> } } };

export type AdminUpdateNoteGraphPositionsMutationVariables = Exact<{
  input: UpdateNoteGraphPositionsInput;
}>;


export type AdminUpdateNoteGraphPositionsMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'UpdateNoteGraphPositionsPayload', success: boolean, updatedNoteViews: Array<{ __typename?: 'NoteView', id: string, pathId: any, title: string }> } } };

export type AdminSelectNoteViewQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminSelectNoteViewQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allLatestNoteViews: { __typename?: 'AdminLatestNoteViewsConnection', nodes: Array<{ __typename?: 'NoteView', versionId: any, path: string, title: string }> } } };

export type AdminNoteWarningsQueryVariables = Exact<{
  filter?: InputMaybe<AdminLatestNoteViewsFilter>;
}>;


export type AdminNoteWarningsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allLatestNoteViews: { __typename?: 'AdminLatestNoteViewsConnection', nodes: Array<{ __typename?: 'NoteView', id: string, path: string, warnings: Array<{ __typename?: 'NoteWarning', level: NoteWarningLevelEnum, message: string }> }> } } };

export type AdminResetNotFoundPathMutationVariables = Exact<{
  input: ResetNotFoundPathInput;
}>;


export type AdminResetNotFoundPathMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'ResetNotFoundPathPayload', notFoundPath: { __typename: 'AdminNotFoundPath', id: any } } } };

export type AdminNotFoundPathsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminNotFoundPathsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allNotFoundPaths: { __typename?: 'AdminNotFoundPathsConnection', nodes: Array<{ __typename?: 'AdminNotFoundPath', id: any, path: string, totalHits: any, lastHitAt: any }> } } };

export type AdminShowNotFoundPathQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminShowNotFoundPathQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allNotFoundPaths: { __typename?: 'AdminNotFoundPathsConnection', nodes: Array<{ __typename?: 'AdminNotFoundPath', id: any, path: string, totalHits: any, lastHitAt: any }> } } };

export type AdminDeleteNotFoundIgnoredPatternMutationVariables = Exact<{
  input: DeleteNotFoundIgnoredPatternInput;
}>;


export type AdminDeleteNotFoundIgnoredPatternMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename: 'DeleteNotFoundIgnoredPatternPayload', deletedId: any } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminNotFoundIgnoredPatternsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminNotFoundIgnoredPatternsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allNotFoundIgnoredPatterns: { __typename?: 'AdminNotFoundIgnoredPatternsConnection', nodes: Array<{ __typename?: 'AdminNotFoundIgnoredPattern', id: any, pattern: string, createdAt: any, createdBy: { __typename?: 'AdminUser', id: any, email?: string | null } }> } } };

export type AdminCreateNotFoundIgnoredPatternMutationMutationVariables = Exact<{
  input: CreateNotFoundIgnoredPatternInput;
}>;


export type AdminCreateNotFoundIgnoredPatternMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'CreateNotFoundIgnoredPatternPayload', notFoundIgnoredPattern: { __typename?: 'AdminNotFoundIgnoredPattern', id: any } } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminShowNotFoundIgnoredPatternQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminShowNotFoundIgnoredPatternQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allNotFoundIgnoredPatterns: { __typename?: 'AdminNotFoundIgnoredPatternsConnection', nodes: Array<{ __typename?: 'AdminNotFoundIgnoredPattern', id: any, pattern: string, createdAt: any, createdBy: { __typename?: 'AdminUser', id: any, email?: string | null } }> } } };

export type AdminDeleteNotFoundIgnoredPatternMutationMutationVariables = Exact<{
  input: DeleteNotFoundIgnoredPatternInput;
}>;


export type AdminDeleteNotFoundIgnoredPatternMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'DeleteNotFoundIgnoredPatternPayload', deletedId: any } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminUpdateNotFoundIgnoredPatternMutationMutationVariables = Exact<{
  input: UpdateNotFoundIgnoredPatternInput;
}>;


export type AdminUpdateNotFoundIgnoredPatternMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'UpdateNotFoundIgnoredPatternPayload', notFoundIgnoredPattern: { __typename?: 'AdminNotFoundIgnoredPattern', id: any } } } };

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

export type AdminDeletePatreonCredentialsMutationVariables = Exact<{
  input: DeletePatreonCredentialsInput;
}>;


export type AdminDeletePatreonCredentialsMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', deletePatreonCredentials: { __typename?: 'DeletePatreonCredentialsPayload', patreonCredentials: { __typename?: 'AdminPatreonCredentials', id: any } } | { __typename?: 'ErrorPayload', message: string } } };

export type RefreshPatreonDataMutationVariables = Exact<{
  input: RefreshPatreonDataInput;
}>;


export type RefreshPatreonDataMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', refreshPatreonData: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'RefreshPatreonDataPayload', success: boolean } } };

export type AdminRestorePatreonCredentialsMutationVariables = Exact<{
  input: RestorePatreonCredentialsInput;
}>;


export type AdminRestorePatreonCredentialsMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', restorePatreonCredentials: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'RestorePatreonCredentialsPayload', patreonCredentials: { __typename?: 'AdminPatreonCredentials', id: any } } } };

export type AdminPatreonCredentialsQueryVariables = Exact<{
  filter?: InputMaybe<AdminPatreonCredentialsFilterInput>;
}>;


export type AdminPatreonCredentialsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allPatreonCredentials: { __typename?: 'AdminPatreonCredentialsConnection', nodes: Array<{ __typename?: 'AdminPatreonCredentials', id: any, state: PatreonCredentialsStateEnum, creatorAccessToken: string, createdAt: any, syncedAt?: any | null, createdBy: { __typename?: 'AdminUser', id: any, email?: string | null } }> } } };

export type AdminCreatePatreonCredsMutationVariables = Exact<{
  input: CreatePatreonCredentialsInput;
}>;


export type AdminCreatePatreonCredsMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', createPatreonCredentials: { __typename?: 'CreatePatreonCredentialsPayload', patreonCredentials: { __typename?: 'AdminPatreonCredentials', id: any } } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminPatreonCredentialsByIdQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminPatreonCredentialsByIdQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', patreonCredentials?: { __typename?: 'AdminPatreonCredentials', createdAt: any, creatorAccessToken: string, state: PatreonCredentialsStateEnum, createdBy: { __typename?: 'AdminUser', email?: string | null }, tiers: { __typename?: 'AdminPatreonTiersConnection', nodes: Array<{ __typename?: 'AdminPatreonTier', id: any, missedAt?: any | null, title: string, amountCents: any, subgraphs: Array<{ __typename?: 'AdminSubgraph', id: any }> }> }, members: { __typename?: 'AdminPatreonMembersConnection', nodes: Array<{ __typename?: 'AdminPatreonMember', email: string, status: string, currentTier?: { __typename?: 'AdminPatreonTier', title: string } | null }> } } | null } };

export type AdminPatreoncredentialsShowSubgraphsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminPatreoncredentialsShowSubgraphsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename?: 'AdminSubgraph', id: any, name: string }> } } };

export type AdminPatreoncredentialsShowSubgraphsSaveMutationVariables = Exact<{
  input: SetPatreonTierSubgraphsInput;
}>;


export type AdminPatreoncredentialsShowSubgraphsSaveMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'SetPatreonTierSubgraphsPayload', success: boolean } } };

export type AdminPurchasesQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminPurchasesQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allPurchases: { __typename?: 'AdminPurchasesConnection', nodes: Array<{ __typename?: 'AdminPurchase', id: string, createdAt: any, paymentProvider: string, status: string, successful: boolean, offerId: any, email: string }> } } };

export type AdminRedirectsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminRedirectsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allRedirects: { __typename?: 'AdminRedirectsConnection', nodes: Array<{ __typename?: 'AdminRedirect', id: any, createdAt: any, pattern: string, ignoreCase: boolean, isRegex: boolean, target: string, createdBy: { __typename?: 'AdminUser', id: any, email?: string | null } }> } } };

export type AdminCreateRedirectMutationMutationVariables = Exact<{
  input: CreateRedirectInput;
}>;


export type AdminCreateRedirectMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'CreateRedirectPayload', redirect: { __typename?: 'AdminRedirect', id: any } } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminShowRedirectQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminShowRedirectQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', redirect?: { __typename?: 'AdminRedirect', id: any, createdAt: any, pattern: string, ignoreCase: boolean, isRegex: boolean, target: string, createdBy: { __typename?: 'AdminUser', id: any, email?: string | null } } | null } };

export type AdminDeleteRedirectMutationMutationVariables = Exact<{
  input: DeleteRedirectInput;
}>;


export type AdminDeleteRedirectMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'DeleteRedirectPayload', id: any } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminUpdateRedirectMutationMutationVariables = Exact<{
  input: UpdateRedirectInput;
}>;


export type AdminUpdateRedirectMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'UpdateRedirectPayload', redirect: { __typename?: 'AdminRedirect', id: any } } } };

export type AdminMakeReleaseLiveMutationVariables = Exact<{
  input: MakeReleaseLiveInput;
}>;


export type AdminMakeReleaseLiveMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'MakeReleaseLivePayload', release: { __typename?: 'AdminRelease', id: any } } } };

export type AdminReleasesQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminReleasesQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allReleases: { __typename?: 'AdminReleasesConnection', nodes: Array<{ __typename?: 'AdminRelease', id: any, createdAt: any, title: string, isLive: boolean, createdBy: { __typename?: 'AdminUser', email?: string | null } }> } } };

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


export type AdminShowSubgraphQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', subgraph?: { __typename?: 'AdminSubgraph', id: any, name: string, color?: string | null, hidden: boolean } | null } };

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

export type AdminTgBotsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminTgBotsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allTgBots: { __typename?: 'AdminTgBotsConnection', nodes: Array<{ __typename?: 'AdminTgBot', id: any, name: string, description: string, enabled: boolean, createdAt: any, createdBy: { __typename?: 'AdminUser', email?: string | null } }> } } };

export type AdminCreateTgBotMutationMutationVariables = Exact<{
  input: CreateTgBotInput;
}>;


export type AdminCreateTgBotMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'CreateTgBotPayload', tgBot: { __typename?: 'AdminTgBot', id: any, name: string } } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminTgBotChatsQueryVariables = Exact<{
  filter: AdminTgBotChatsFilterInput;
}>;


export type AdminTgBotChatsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', tgBotChats: { __typename?: 'AdminTgBotChatsConnection', nodes: Array<{ __typename?: 'AdminTgBotChat', id: any, chatType: string, chatTitle: string, addedAt: any, removedAt?: any | null, memberCount: number, subgraphAccesses: Array<{ __typename?: 'AdminTgChatSubgraphAccess', id: any, subgraphId: any }> }> } } };

export type AdminTgbotsShowchatsSubgraphsSaveMutationVariables = Exact<{
  input: SetTgChatSubgraphsInput;
}>;


export type AdminTgbotsShowchatsSubgraphsSaveMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'SetTgChatSubgraphsPayload', success: boolean } } };

export type AdminShowTgBotQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminShowTgBotQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', tgBot?: { __typename?: 'AdminTgBot', id: any, name: string, description: string, enabled: boolean, createdAt: any, createdBy: { __typename?: 'AdminUser', email?: string | null } } | null } };

export type AdminUpdateTgBotMutationMutationVariables = Exact<{
  input: UpdateTgBotInput;
}>;


export type AdminUpdateTgBotMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'UpdateTgBotPayload', tgBot: { __typename?: 'AdminTgBot', id: any, description: string } } } };

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


export type ViewerQuery = { __typename?: 'Query', viewer: { __typename?: 'Viewer', id: string, role: Role, user?: { __typename?: 'User', email?: string | null } | null } };

export type ReaderQueryQueryVariables = Exact<{
  input: NoteInput;
}>;


export type ReaderQueryQuery = { __typename?: 'Query', note?: { __typename?: 'PublicNote', title: string, html: string } | null };

export type PaywallActivePurchaseQueryQueryVariables = Exact<{ [key: string]: never; }>;


export type PaywallActivePurchaseQueryQuery = { __typename?: 'Query', viewer: { __typename?: 'Viewer', activePurchases: Array<{ __typename?: 'Purchase', id: string, status: string, successful: boolean }> } };

export type CreateEmailWaitListRequestMutationMutationVariables = Exact<{
  input: CreateEmailWaitListRequestInput;
}>;


export type CreateEmailWaitListRequestMutationMutation = { __typename?: 'Mutation', createEmailWaitListRequest: { __typename?: 'CreateEmailWaitListRequestPayload', success: boolean } | { __typename?: 'ErrorPayload', message: string } };

export type PaywallQueryQueryVariables = Exact<{
  filter: ViewerOffersFilter;
}>;


export type PaywallQueryQuery = { __typename?: 'Query', viewer: { __typename?: 'Viewer', offers?: { __typename?: 'ActiveOffers', nodes: Array<{ __typename?: 'Offer', id: string, priceUSD: number, subgraphs: Array<{ __typename?: 'Subgraph', name: string }> }> } | { __typename?: 'SubgraphWaitList', tgBotUrl?: string | null, emailAllowed: boolean } | null } };

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

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminDeleteBoostyCredentials($input: DeleteBoostyCredentialsInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdeleteBoostyCredentials(input: $input) {\n\t\t\t\t\t\t\t\t... on ErrorPayload{\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on DeleteBoostyCredentialsPayload {\n\t\t\t\t\t\t\t\t\tboostyCredentials {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminDeleteBoostyCredentialsMutationVariables): AdminDeleteBoostyCredentialsMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation RefreshBoostyData($input: RefreshBoostyDataInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\trefreshBoostyData(input: $input) {\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on RefreshBoostyDataPayload {\n\t\t\t\t\t\t\t\t\tsuccess\n\t\t\t\t\t\t\t\t\tcredentials {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: RefreshBoostyDataMutationVariables): RefreshBoostyDataMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminRestoreBoostyCredentials($input: RestoreBoostyCredentialsInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\trestoreBoostyCredentials(input: $input) {\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on RestoreBoostyCredentialsPayload {\n\t\t\t\t\t\t\t\t\tboostyCredentials {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminRestoreBoostyCredentialsMutationVariables): AdminRestoreBoostyCredentialsMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminBoostyCredentials($filter: AdminBoostyCredentialsFilterInput) {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallBoostyCredentials(filter: $filter) {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tstate\n\t\t\t\t\t\t\t\tdeviceId\n\t\t\t\t\t\t\t\tblogName\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t', variables: AdminBoostyCredentialsQueryVariables): AdminBoostyCredentialsQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminCreateBoostyCreds($input: CreateBoostyCredentialsInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tcreateBoostyCredentials(input: $input) {\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on CreateBoostyCredentialsPayload {\n\t\t\t\t\t\t\t\t\tboostyCredentials {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminCreateBoostyCredsMutationVariables): AdminCreateBoostyCredsMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminBoostyCredentialsById($id: Int64!) {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tboostyCredentials(id: $id) {\n\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\tdeviceId\n\t\t\t\t\t\t\tblogName\n\t\t\t\t\t\t\tstate\n\n\t\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t}\n\n\t\t\t\t\t\t\ttiers {\n\t\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\tname\n\n\t\t\t\t\t\t\t\t\tsubgraphs {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\n\t\t\t\t\t\t\tmembers {\n\t\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t\tstatus\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t', variables: AdminBoostyCredentialsByIdQueryVariables): AdminBoostyCredentialsByIdQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminBoostycredentialsShowSubgraphs {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tallSubgraphs {\n\t\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t'): AdminBoostycredentialsShowSubgraphsQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tmutation AdminBoostycredentialsShowSubgraphsSave($input: SetBoostyTierSubgraphsInput!) {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tdata: setBoostyTierSubgraphs(input: $input) {\n\t\t\t\t\t\t\t... on SetBoostyTierSubgraphsPayload {\n\t\t\t\t\t\t\t\tsuccess\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t', variables: AdminBoostycredentialsShowSubgraphsSaveMutationVariables): AdminBoostycredentialsShowSubgraphsSaveMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tmutation AdminUnbanUser($input: UnbanUserInput!) {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tdata: unbanUser(input: $input) {\n\t\t\t\t\t\t\t... on UnbanUserPayload {\n\t\t\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\n\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t', variables: AdminUnbanUserMutationVariables): AdminUnbanUserMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminListNoteViews {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallLatestNoteViews {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tpath\n\t\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t\t\tfree\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminListNoteViewsQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminListSubgraphs {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallSubgraphs {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\tcolor\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminListSubgraphsQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminListUserBans {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallUserUserBans {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid: userId\n\t\t\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\tbannedBy {\n\t\t\t\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\treason\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminListUserBansQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminListUsers {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallUsers {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\tban { reason }\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminListUsersQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminListUserSubgraphAccesses {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tdata: allUserSubgraphAccesses {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\texpiresAt\n\t\t\t\t\t\t\t\tsubgraph {\n\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminListUserSubgraphAccessesQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminGraph {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallSubgraphs {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\tcolor\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t\tallLatestNoteViews {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tsubgraphNames\n\t\t\t\t\t\t\t\tpathId\n\t\t\t\t\t\t\t\tfree\n\t\t\t\t\t\t\t\tisHomePage\n\t\t\t\t\t\t\t\tgraphPosition{\n\t\t\t\t\t\t\t\t\tx,\n\t\t\t\t\t\t\t\t\ty,\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\tinLinks {\n\t\t\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t\t\t\tpathId\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminGraphQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminUpdateNoteGraphPositions($input: UpdateNoteGraphPositionsInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: updateNoteGraphPositions(input: $input) {\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on UpdateNoteGraphPositionsPayload {\n\t\t\t\t\t\t\t\t\tsuccess\n\t\t\t\t\t\t\t\t\tupdatedNoteViews {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t\tpathId\n\t\t\t\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminUpdateNoteGraphPositionsMutationVariables): AdminUpdateNoteGraphPositionsMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminSelectNoteView {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tallLatestNoteViews {\n\t\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t\tversionId\n\t\t\t\t\t\t\t\t\tpath\n\t\t\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t'): AdminSelectNoteViewQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminNoteWarnings($filter: AdminLatestNoteViewsFilter) {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallLatestNoteViews(filter: $filter) {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tpath\n\t\t\t\t\t\t\t\twarnings {\n\t\t\t\t\t\t\t\t\tlevel\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t', variables: AdminNoteWarningsQueryVariables): AdminNoteWarningsQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tmutation AdminResetNotFoundPath($input: ResetNotFoundPathInput!) {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tdata: resetNotFoundPath(input: $input) {\n\t\t\t\t\t\t\t... on ResetNotFoundPathPayload {\n\t\t\t\t\t\t\t\tnotFoundPath {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\n\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t', variables: AdminResetNotFoundPathMutationVariables): AdminResetNotFoundPathMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminNotFoundPaths {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallNotFoundPaths {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tpath\n\t\t\t\t\t\t\t\ttotalHits\n\t\t\t\t\t\t\t\tlastHitAt\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminNotFoundPathsQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminShowNotFoundPath {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tallNotFoundPaths {\n\t\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\tpath\n\t\t\t\t\t\t\t\t\ttotalHits\n\t\t\t\t\t\t\t\t\tlastHitAt\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t'): AdminShowNotFoundPathQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tmutation AdminDeleteNotFoundIgnoredPattern($input: DeleteNotFoundIgnoredPatternInput!) {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tdata: deleteNotFoundIgnoredPattern(input: $input) {\n\t\t\t\t\t\t\t... on DeleteNotFoundIgnoredPatternPayload {\n\t\t\t\t\t\t\t\tdeletedId\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t}\n\n\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t', variables: AdminDeleteNotFoundIgnoredPatternMutationVariables): AdminDeleteNotFoundIgnoredPatternMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminNotFoundIgnoredPatterns {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallNotFoundIgnoredPatterns {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tpattern\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminNotFoundIgnoredPatternsQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminCreateNotFoundIgnoredPatternMutation($input: CreateNotFoundIgnoredPatternInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: createNotFoundIgnoredPattern(input: $input) {\n\t\t\t\t\t\t\t\t... on CreateNotFoundIgnoredPatternPayload {\n\t\t\t\t\t\t\t\t\tnotFoundIgnoredPattern {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminCreateNotFoundIgnoredPatternMutationMutationVariables): AdminCreateNotFoundIgnoredPatternMutationMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminShowNotFoundIgnoredPattern {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tallNotFoundIgnoredPatterns {\n\t\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\tpattern\n\t\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t'): AdminShowNotFoundIgnoredPatternQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminDeleteNotFoundIgnoredPatternMutation($input: DeleteNotFoundIgnoredPatternInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: deleteNotFoundIgnoredPattern(input: $input) {\n\t\t\t\t\t\t\t\t... on DeleteNotFoundIgnoredPatternPayload {\n\t\t\t\t\t\t\t\t\tdeletedId\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminDeleteNotFoundIgnoredPatternMutationMutationVariables): AdminDeleteNotFoundIgnoredPatternMutationMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminUpdateNotFoundIgnoredPatternMutation($input: UpdateNotFoundIgnoredPatternInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: updateNotFoundIgnoredPattern(input: $input) {\n\t\t\t\t\t\t\t\t... on UpdateNotFoundIgnoredPatternPayload {\n\t\t\t\t\t\t\t\t\tnotFoundIgnoredPattern {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminUpdateNotFoundIgnoredPatternMutationMutationVariables): AdminUpdateNotFoundIgnoredPatternMutationMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminOffers {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallOffers {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tpublicId\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\tlifetime\n\t\t\t\t\t\t\t\tpriceUSD\n\t\t\t\t\t\t\t\tstartsAt\n\t\t\t\t\t\t\t\tendsAt\n\t\t\t\t\t\t\t\tsubgraphs {\n\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminOffersQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminCreateOfferMutation($input: CreateOfferInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: createOffer(input: $input) {\n\t\t\t\t\t\t\t\t... on CreateOfferPayload {\n\t\t\t\t\t\t\t\t\toffer {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t\tpublicId\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminCreateOfferMutationMutationVariables): AdminCreateOfferMutationMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminShowOffer($id: Int64!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\toffer(id: $id) {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tpublicId\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\tlifetime\n\t\t\t\t\t\t\t\tpriceUSD\n\t\t\t\t\t\t\t\tstartsAt\n\t\t\t\t\t\t\t\tendsAt\n\t\t\t\t\t\t\t\tsubgraphIds\n\t\t\t\t\t\t\t\tsubgraphs {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminShowOfferQueryVariables): AdminShowOfferQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminUpdateOfferMutation($input: UpdateOfferInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: updateOffer(input: $input) {\n\t\t\t\t\t\t\t\t... on UpdateOfferPayload {\n\t\t\t\t\t\t\t\t\toffer {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t\tpublicId\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminUpdateOfferMutationMutationVariables): AdminUpdateOfferMutationMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminDeletePatreonCredentials($input: DeletePatreonCredentialsInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdeletePatreonCredentials(input: $input) {\n\t\t\t\t\t\t\t\t... on ErrorPayload{\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on DeletePatreonCredentialsPayload {\n\t\t\t\t\t\t\t\t\tpatreonCredentials {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminDeletePatreonCredentialsMutationVariables): AdminDeletePatreonCredentialsMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation RefreshPatreonData($input: RefreshPatreonDataInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\trefreshPatreonData(input: $input) {\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on RefreshPatreonDataPayload {\n\t\t\t\t\t\t\t\t\tsuccess\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: RefreshPatreonDataMutationVariables): RefreshPatreonDataMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminRestorePatreonCredentials($input: RestorePatreonCredentialsInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\trestorePatreonCredentials(input: $input) {\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on RestorePatreonCredentialsPayload {\n\t\t\t\t\t\t\t\t\tpatreonCredentials {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminRestorePatreonCredentialsMutationVariables): AdminRestorePatreonCredentialsMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminPatreonCredentials($filter: AdminPatreonCredentialsFilterInput) {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallPatreonCredentials(filter: $filter) {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tstate\n\t\t\t\t\t\t\t\tcreatorAccessToken\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\tsyncedAt\n\t\t\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t', variables: AdminPatreonCredentialsQueryVariables): AdminPatreonCredentialsQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminCreatePatreonCreds($input: CreatePatreonCredentialsInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tcreatePatreonCredentials(input: $input) {\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on CreatePatreonCredentialsPayload {\n\t\t\t\t\t\t\t\t\tpatreonCredentials {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminCreatePatreonCredsMutationVariables): AdminCreatePatreonCredsMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminPatreonCredentialsById($id: Int64!) {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tpatreonCredentials(id: $id) {\n\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\tcreatorAccessToken\n\t\t\t\t\t\t\tstate\n\n\t\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t}\n\n\t\t\t\t\t\t\ttiers {\n\t\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\tmissedAt\n\t\t\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t\t\t\tamountCents\n\n\t\t\t\t\t\t\t\t\tsubgraphs {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\n\t\t\t\t\t\t\tmembers {\n\t\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t\tstatus\n\t\t\t\t\t\t\t\t\tcurrentTier {\n\t\t\t\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t', variables: AdminPatreonCredentialsByIdQueryVariables): AdminPatreonCredentialsByIdQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminPatreoncredentialsShowSubgraphs {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tallSubgraphs {\n\t\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t'): AdminPatreoncredentialsShowSubgraphsQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tmutation AdminPatreoncredentialsShowSubgraphsSave($input: SetPatreonTierSubgraphsInput!) {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tdata: setPatreonTierSubgraphs(input: $input) {\n\t\t\t\t\t\t\t... on SetPatreonTierSubgraphsPayload {\n\t\t\t\t\t\t\t\tsuccess\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t', variables: AdminPatreoncredentialsShowSubgraphsSaveMutationVariables): AdminPatreoncredentialsShowSubgraphsSaveMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminPurchases {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallPurchases {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\tpaymentProvider\n\t\t\t\t\t\t\t\tstatus\n\t\t\t\t\t\t\t\tsuccessful\n\t\t\t\t\t\t\t\tofferId\n\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminPurchasesQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminRedirects {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallRedirects {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\tpattern\n\t\t\t\t\t\t\t\tignoreCase\n\t\t\t\t\t\t\t\tisRegex\n\t\t\t\t\t\t\t\ttarget\n\t\t\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminRedirectsQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminCreateRedirectMutation($input: CreateRedirectInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: createRedirect(input: $input) {\n\t\t\t\t\t\t\t\t... on CreateRedirectPayload {\n\t\t\t\t\t\t\t\t\tredirect {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminCreateRedirectMutationMutationVariables): AdminCreateRedirectMutationMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminShowRedirect($id: Int64!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tredirect(id: $id) {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\tpattern\n\t\t\t\t\t\t\t\tignoreCase\n\t\t\t\t\t\t\t\tisRegex\n\t\t\t\t\t\t\t\ttarget\n\t\t\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminShowRedirectQueryVariables): AdminShowRedirectQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminDeleteRedirectMutation($input: DeleteRedirectInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: deleteRedirect(input: $input) {\n\t\t\t\t\t\t\t\t... on DeleteRedirectPayload {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminDeleteRedirectMutationMutationVariables): AdminDeleteRedirectMutationMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminUpdateRedirectMutation($input: UpdateRedirectInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: updateRedirect(input: $input) {\n\t\t\t\t\t\t\t\t... on UpdateRedirectPayload {\n\t\t\t\t\t\t\t\t\tredirect {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminUpdateRedirectMutationMutationVariables): AdminUpdateRedirectMutationMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminMakeReleaseLive($input: MakeReleaseLiveInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: makeReleaseLive(input:$input) {\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on MakeReleaseLivePayload {\n\t\t\t\t\t\t\t\t\trelease {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminMakeReleaseLiveMutationVariables): AdminMakeReleaseLiveMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminReleases {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallReleases {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\tcreatedBy{\n\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t\t\tisLive\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminReleasesQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminCreateRelease($input: CreateReleaseInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: createRelease(input: $input) {\n\t\t\t\t\t\t\t\t... on CreateReleasePayload {\n\t\t\t\t\t\t\t\t\trelease {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminCreateReleaseMutationVariables): AdminCreateReleaseMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tmutation AdminBanUser($input: BanUserInput!) {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tbanUser(input: $input) {\n\t\t\t\t\t\t\t... on BanUserPayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tuser { id, __typename }\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t', variables: AdminBanUserMutationVariables): AdminBanUserMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminNoteView($id: String!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tnoteView(id: $id) {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tpath\n\t\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t\t\tpermalink\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminNoteViewQueryVariables): AdminNoteViewQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminShowSubgraph($id: Int64!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tsubgraph(id: $id) {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\tcolor\n\t\t\t\t\t\t\t\thidden\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminShowSubgraphQueryVariables): AdminShowSubgraphQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation UpdateSubgraph($input: UpdateSubgraphInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: updateSubgraph(input: $input) {\n\t\t\t\t\t\t\t\t... on UpdateSubgraphPayload {\n\t\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\t\tsubgraph {\n\t\t\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t\tcolor\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: UpdateSubgraphMutationVariables): UpdateSubgraphMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminUserSubgraphAccess($id: Int64!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tallSubgraphs {\n\t\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\n\t\t\t\t\t\t\tuserSubgraphAccess(id: $id) {\n\t\t\t\t\t\t\t\tuserId\n\t\t\t\t\t\t\t\tsubgraphId\n\t\t\t\t\t\t\t\texpiresAt\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminUserSubgraphAccessQueryVariables): AdminUserSubgraphAccessQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminUpdateUserSubgraphAccess($input: UpdateUserSubgraphAccessInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: updateUserSubgraphAccess(input: $input) {\n\t\t\t\t\t\t\t\t... on UpdateUserSubgraphAccessPayload {\n\t\t\t\t\t\t\t\t\tuserSubgraphAccess {\n\t\t\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\t\t\texpiresAt\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminUpdateUserSubgraphAccessMutationVariables): AdminUpdateUserSubgraphAccessMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminSelectSubgraphList {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tallSubgraphs {\n\t\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t'): AdminSelectSubgraphListQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminSelectSubgraph {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tallSubgraphs {\n\t\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t'): AdminSelectSubgraphQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminTgBots {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tallTgBots {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\tdescription\n\t\t\t\t\t\t\t\tenabled\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): AdminTgBotsQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminCreateTgBotMutation($input: CreateTgBotInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: createTgBot(input: $input) {\n\t\t\t\t\t\t\t\t... on CreateTgBotPayload {\n\t\t\t\t\t\t\t\t\ttgBot {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminCreateTgBotMutationMutationVariables): AdminCreateTgBotMutationMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery AdminTgBotChats($filter: AdminTgBotChatsFilterInput!) {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\ttgBotChats(filter: $filter) {\n\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tchatType\n\t\t\t\t\t\t\t\tchatTitle\n\t\t\t\t\t\t\t\taddedAt\n\t\t\t\t\t\t\t\tremovedAt\n\t\t\t\t\t\t\t\tmemberCount\n\t\t\t\t\t\t\t\tsubgraphAccesses {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\tsubgraphId\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t', variables: AdminTgBotChatsQueryVariables): AdminTgBotChatsQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tmutation AdminTgbotsShowchatsSubgraphsSave($input: SetTgChatSubgraphsInput!) {\n\t\t\t\t\tadmin {\n\t\t\t\t\t\tdata: setTgChatSubgraphs(input: $input) {\n\t\t\t\t\t\t\t... on SetTgChatSubgraphsPayload {\n\t\t\t\t\t\t\t\tsuccess\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t', variables: AdminTgbotsShowchatsSubgraphsSaveMutationVariables): AdminTgbotsShowchatsSubgraphsSaveMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tquery AdminShowTgBot($id: Int64!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\ttgBot(id: $id) {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\tdescription\n\t\t\t\t\t\t\t\tenabled\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminShowTgBotQueryVariables): AdminShowTgBotQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation AdminUpdateTgBotMutation($input: UpdateTgBotInput!) {\n\t\t\t\t\t\tadmin {\n\t\t\t\t\t\t\tdata: updateTgBot(input: $input) {\n\t\t\t\t\t\t\t\t... on UpdateTgBotPayload {\n\t\t\t\t\t\t\t\t\ttgBot {\n\t\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\t\tdescription\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: AdminUpdateTgBotMutationMutationVariables): AdminUpdateTgBotMutationMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tmutation SignOut {\n\t\t\t\t\tdata: signOut {\n\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t\t... on SignOutPayload {\n\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\tviewer {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): SignOutMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation RequestEmailSignInCode($input: RequestEmailSignInCodeInput!) {\n\t\t\t\t\t\tdata: requestEmailSignInCode(input: $input) {\n\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t... on RequestEmailSignInCodePayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tsuccess\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: RequestEmailSignInCodeMutationVariables): RequestEmailSignInCodeMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation SignInByEmail($input: SignInByEmailInput!) {\n\t\t\t\t\t\tdata: signInByEmail(input: $input) {\n\t\t\t\t\t\t\t... on SignInPayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\ttoken\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: SignInByEmailMutationVariables): SignInByEmailMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery Viewer {\n\t\t\t\t\tviewer {\n\t\t\t\t\t\tid\n\t\t\t\t\t\trole\n\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): ViewerQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery ReaderQuery($input: NoteInput!) {\n\t\t\t\t\tnote(input: $input) {\n\t\t\t\t\t\ttitle\n\t\t\t\t\t\thtml\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t', variables: ReaderQueryQueryVariables): ReaderQueryQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery PaywallActivePurchaseQuery {\n\t\t\t\t\tviewer {\n\t\t\t\t\t\tactivePurchases {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tstatus\n\t\t\t\t\t\t\tsuccessful\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): PaywallActivePurchaseQueryQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\t\tmutation CreateEmailWaitListRequestMutation ($input: CreateEmailWaitListRequestInput!) {\n\t\t\t\t\t\tcreateEmailWaitListRequest(input: $input) {\n\t\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t... on CreateEmailWaitListRequestPayload {\n\t\t\t\t\t\t\t\tsuccess\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t', variables: CreateEmailWaitListRequestMutationMutationVariables): CreateEmailWaitListRequestMutationMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery PaywallQuery($filter: ViewerOffersFilter!) {\n\t\t\t\t\tviewer {\n\t\t\t\t\t\toffers(filter: $filter) {\n\t\t\t\t\t\t\t... on ActiveOffers {\n\t\t\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\t\tpriceUSD\n\t\t\t\t\t\t\t\t\tsubgraphs {\n\t\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t... on SubgraphWaitList {\n\t\t\t\t\t\t\t\ttgBotUrl\n\t\t\t\t\t\t\t\temailAllowed\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\n\t\t\t\t}\n\t\t\t', variables: PaywallQueryQueryVariables): PaywallQueryQuery

export function $trip2g_graphql_request(query: '\n\t\t\t\tmutation CreatePaymentLink($input: CreatePaymentLinkInput!) {\n\t\t\t\t\tdata: createPaymentLink(input: $input) {\n\t\t\t\t\t\t... on CreatePaymentLinkPayload {\n\t\t\t\t\t\t\tredirectUrl\n\t\t\t\t\t\t}\n\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t\t', variables: CreatePaymentLinkMutationVariables): CreatePaymentLinkMutation

export function $trip2g_graphql_request(query: '\n\t\t\t\tquery UserSubscriptions {\n\t\t\t\t\tviewer {\n\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\tsubgraphAccesses {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\t\texpiresAt\n\t\t\t\t\t\t\t\tsubgraph {\n\t\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t\t\thomePath\n\t\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t'): UserSubscriptionsQuery

export function $trip2g_graphql_request(query: any, variables?: any) { return $trip2g_graphql_raw_request(query, variables); }

export function $trip2g_graphql_subscription(query: any, variables?: any) { return $trip2g_graphql_raw_subscription(query, variables); }



export const $trip2g_graphql_persist_queries = {"Admins":"7b9f99a6b0b785b43488198eb4dec442d88e4abda7c516677a0611d76878d904","DisableApiKey":"a5852655edeb09cf7db15b0a196cc53cc005b3f8ef7fe7d7d3cdc032087b8b6b","AdminListApiKeys":"1baa27852f59c95f35fe5f35635ad6a617fca6312411bd6cae7cd377d681a52d","AdminCreateApiKey":"c9c10cfb6fa133ac380870427e9b9cdebbc1e9e9b92e6a6313c97f52b537d66f","AdminApiKeyShowQuery":"2f8b56fce14a35ac51fd315e919a352048a118b036a0368f2a66a85c7156c24b","AdminDeleteBoostyCredentials":"b5b95823f1ca72f823a7e5d185985c7866cfdd4e3e047df82df3e874fd4ded9f","RefreshBoostyData":"6f56dc12aa5527c4b3d9a258d6edbc419df8f5d363e5fef8d6fff9ee2fbada97","AdminRestoreBoostyCredentials":"c4c8090f6a9d499fc2af881e74bafe9311cef92a6d27d9d5d4f77c71a9a7e578","AdminBoostyCredentials":"d85c51db210ba2829aa9f078929fc8b35fab8b93ff4e65ea3cfe4899239c33aa","AdminCreateBoostyCreds":"f4b60eb1f2085c4177c50fd13d76afda3386af6ac3265a223b1b4e25f6fc6eff","AdminBoostyCredentialsById":"b6299c8cfea9f51bdfa87c8d889b099cc2ce517eeed41d852b8e572c35784c0d","AdminBoostycredentialsShowSubgraphs":"91b4086778db021cd991ba8d0b9244077f217e0fd9912857ee8207d19b3597e1","AdminBoostycredentialsShowSubgraphsSave":"1b95680f24795df02b1da6ad640ca527a88b2fbf6cdaa6928bdab79d4472a85d","AdminUnbanUser":"9512bb945535dd9ef2fe1dacc60073ce01fc8d8f3e93fec6583ab2dc455d1309","AdminListNoteViews":"08631c2621fdb1e1265d238476428a5a108673be31c9fa3823233984adaee8ea","AdminListSubgraphs":"4e45ae80a24576cab70fdbb4790a0a7acbde1171823623ed8d7a00e495b596cd","AdminListUserBans":"69e1b4b4cb152647fa3474d44d1fccc7e5bfdc9bfa95d60b9726d4e52c94b3b4","AdminListUsers":"6f4fcb27423e59a080c8c0ba8cc8b69628bf9f961bdfbce5c62bf60c19db4075","AdminListUserSubgraphAccesses":"79ca5aafd82b91a579c3ea6232fa7a32773f2b74846785a00e0f0deee9854eca","AdminGraph":"39e949dd3c2f89603f0d09f5e348c2f46ee78f28d2412ff6efa9279ab68e457b","AdminUpdateNoteGraphPositions":"79055eb93ba30f15dc82d77bdce102ad7d30a32b24e57ebf3339507d9fc66fee","AdminSelectNoteView":"d158e0da61a2cbad548f8ada69595f5f5534f15275a06ef95b969d89a7edb3ad","AdminNoteWarnings":"e1626da4e2828f1a95a9a55f70528b7074352fe80ea42e51a74bd6c695c01d37","AdminResetNotFoundPath":"1f10e8c2c12124d4932f5545eccbe0a2dede0dbb988cf0871f847d8d4e067d1a","AdminNotFoundPaths":"5cf3ec8aa2cf4d233d95d80889f90bd4d8cca6d0f4ea18f149dfbc1ff4fd3281","AdminShowNotFoundPath":"8ebf2f3ec53e223c367e1f99010372c46035e12533946751b5a63abf9bea4fd5","AdminDeleteNotFoundIgnoredPattern":"920a973936047b6abbe9e08334afc4334d661edda95a0bea16925a22524ad6a8","AdminNotFoundIgnoredPatterns":"429f371229ee3f9f65a6bd832b98d12ac2672d4b621c560399c89d30d0f605f0","AdminCreateNotFoundIgnoredPatternMutation":"9748b51ba96309a4b5b2905313c2fc9a8e75f7cdbb0b67a1b4c29884d23ec8c0","AdminShowNotFoundIgnoredPattern":"d2f0732335182b898725504e1f9b01cb043034a550686d13e0c1e5c6ab863344","AdminDeleteNotFoundIgnoredPatternMutation":"55537b7c7f93d3c0b32751d112c5545ca2b45c93261aa804fa154997e9ad2f8e","AdminUpdateNotFoundIgnoredPatternMutation":"f4e434af1218ee8662ed41a5ddee1043ea4fd46858cce1b9d03b87456a5d26be","AdminOffers":"b2fc36437eb7fc9ce54ed7fe52c5d6357eb56c1a2a739e52ed5a355499a817ce","AdminCreateOfferMutation":"54bfc14494b6645090140ee5379fef55991e303aa9bd6df0a022bb6b7285297f","AdminShowOffer":"644549579b7660594fa585b864956004f5a0ff8117877656f5089df760c1d79d","AdminUpdateOfferMutation":"bad9b0a25d99a5b3c45ca5116f543b11b7911d1c8e086ff532efd53f29908ca6","AdminDeletePatreonCredentials":"0e64c279c079b1d77334115113bde43b3fb67c87f2910e3ec78b665711f8e3a9","RefreshPatreonData":"7f0600138abbc345fe5bc1fc3b9b3baab2c1cc26f7aa93adee98bdc3155c5b7d","AdminRestorePatreonCredentials":"5d13761fba27efff289309ffd9a8b5db017c28f87955b985a8c9db1aa9601690","AdminPatreonCredentials":"a4529aa3cd3083e2e27d709dc6b6fa1844d29af82b0d32c29796d920ee3c3f31","AdminCreatePatreonCreds":"591e7cf39ea6e450420cba98196e4442ec0a00116f882fa7f66517b5020ec326","AdminPatreonCredentialsById":"d6d6fd3be69e967017988eb4800eeaba94baafc50e33e35269e8a934dd1313c0","AdminPatreoncredentialsShowSubgraphs":"7907df65e3705a5dee8f06f52c156f65730954cc8a95dd0d07f0d0b98599a548","AdminPatreoncredentialsShowSubgraphsSave":"8190946c3f0341dd41e84850004aa19d425d4aed98ea5984063bb700a46183a6","AdminPurchases":"47a76a00baca079d6e46244a8ba7bf75daf8063fdf530c5a8d46a876fdc00900","AdminRedirects":"2a97bb424ba31abc5de22378e2989fc7c0c4b383de50be424f6f849e831e9311","AdminCreateRedirectMutation":"fb48605b011a659f8c8764b08da9bdadd90cbd5e6bbdf74b1b03d25a58fca89c","AdminShowRedirect":"300e47bb2bb87540db90e9c808e2ddf209f84219236abc6796417bf8bba3fbc0","AdminDeleteRedirectMutation":"b0d9e5d6b48166fd7422947a0b43bf3a93827fb5da666ac04449209b8a8a660c","AdminUpdateRedirectMutation":"70ce4af2a2b9bf672e94cf16d78952c56827591f6c0efb64e1c8e359050da36d","AdminMakeReleaseLive":"d0a1f7de2a4ea212ad0f7b80fd591943f26ca638c29e46acb604f47828db4204","AdminReleases":"a8954143d0ddc89b0a4d8ea0dfc63f3514e085d616d1089f8a0099adc9576993","AdminCreateRelease":"8fa0c84f61a4546c22ab6ccdafb6b18367eaae11e64dd35afeabdcaa9f19ee20","AdminBanUser":"b7eaca2436a0420749fd9818239ddaaf03cf39d13fb051af54e1ddfc16b0a376","AdminNoteView":"71287b914ef21187cc102872ffec82d7d3dbe8f1c7583391227f3829c627cc0d","AdminShowSubgraph":"48c888d8117410ffb10f18fb6676ab2da2a4c9e6a4a83e201a589807632022c6","UpdateSubgraph":"a1ed76b00109cb6eac864a2c57d63d956d4c34676eb519e8d0a33abcd8c8945b","AdminUserSubgraphAccess":"a624b706534097050759fc343d7335c36f60b7fd8cb7eac35979d2c2e0b166da","AdminUpdateUserSubgraphAccess":"1f5c1a927c82b86a3d0e0313744fe51f023ed479f955e6846225bff4ab75a91d","AdminSelectSubgraphList":"bb432284d1aa05d873a33069ba3848e54368d1f8f98ee05fd948722f9a53f92d","AdminSelectSubgraph":"a801d4b303ea060e27e22011e2d7cc74c9a17a95e0877622e55e4012640c11a7","AdminTgBots":"06d210ef2397927fc72a502eb3b74feb72e597821fe1c34225a805aaf3a29d9f","AdminCreateTgBotMutation":"9748fa61fe26e6ae99458cf44e989e723783cd2fc15f15b24cd6fd93270949fa","AdminTgBotChats":"30356ff3696684472ba47034278875cb8b2f3e70b3ca28fa8891695b6c0af067","AdminTgbotsShowchatsSubgraphsSave":"8e1228ade3dd23797937a932335450709f34730a7f0c65c9a9ecc30f34b92954","AdminShowTgBot":"b19db79f571bd11a7ef34fcf55468b5475f00de5ee50426eb09ee71c866af16b","AdminUpdateTgBotMutation":"b52a6b067bf30fd17261a67bc90b1c1805f256483df958579a150b76daddad6b","SignOut":"8e1a898d776a103a20b7dbdfbeb8196fce0175ffe38fa7af9da2848cefdadedf","RequestEmailSignInCode":"cc5a33407c1a3cca08dbcba6847a88298eec84b10f18449244cf13e458449ef9","SignInByEmail":"5678d2ca29b77529e0b9942845dcea2c941c6b5a234863dc86ac2ffaad668614","Viewer":"11c2c89ff045bd18461fa0409d26dcf4acbb063202aa8e74615fcd74d21db756","ReaderQuery":"f7df3b23f1ff47bad46b39d5f1ba0a80a412d236b845497288dd593b3c13d8f6","PaywallActivePurchaseQuery":"49eef6eeabb08d2251039c9b3ca67cade0c7af866c29300712400aa44d34d15b","CreateEmailWaitListRequestMutation":"3239c9d8b52c7fc53db7844e5a7c8c5ee4f262e374d421a695bda48072395718","PaywallQuery":"efbb99141e50ca0ffbc3d9ab51ba259b45601c696bd77ca7334dc6579f88195d","CreatePaymentLink":"3c4ea8c746c573dc9a48a303c13f2b6f7ac3ccad23045f8e989105ea0df98b7a","UserSubscriptions":"1e67f2ab803afe77d6859c2130b2fede9f9edce8c9d841e14ce552c3ecf40115"}

export const $trip2g_graphql_boosty_credentials_state_enum = BoostyCredentialsStateEnum;

export const $trip2g_graphql_note_warning_level_enum = NoteWarningLevelEnum;

export const $trip2g_graphql_patreon_credentials_state_enum = PatreonCredentialsStateEnum;

export const $trip2g_graphql_payment_type = PaymentType;

export const $trip2g_graphql_role = Role;

}