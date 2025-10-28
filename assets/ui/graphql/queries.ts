namespace $ {


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

export type AdminAuditLog = {
  __typename?: 'AdminAuditLog';
  createdAt: Scalars['Time']['output'];
  id: Scalars['Int64']['output'];
  level: AuditLogLevelEnum;
  message: Scalars['String']['output'];
  params: Scalars['String']['output'];
};

export type AdminAuditLogsConnection = {
  __typename?: 'AdminAuditLogsConnection';
  nodes: Array<AdminAuditLog>;
};

export type AdminAuditLogsDateFilter = {
  gte?: InputMaybe<Scalars['Time']['input']>;
  lte?: InputMaybe<Scalars['Time']['input']>;
};

export type AdminAuditLogsFilterInput = {
  createdAt?: InputMaybe<AdminAuditLogsDateFilter>;
  limit?: InputMaybe<Scalars['Int64']['input']>;
  offset?: InputMaybe<Scalars['Int64']['input']>;
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

export type AdminConfigVersion = {
  __typename?: 'AdminConfigVersion';
  createdAt: Scalars['Time']['output'];
  createdBy: AdminUser;
  defaultLayout: Scalars['String']['output'];
  id: Scalars['Int64']['output'];
  showDraftVersions: Scalars['Boolean']['output'];
  timezone: Scalars['String']['output'];
};

export type AdminConfigVersionsConnection = {
  __typename?: 'AdminConfigVersionsConnection';
  nodes: Array<AdminConfigVersion>;
};

export type AdminCronJob = {
  __typename?: 'AdminCronJob';
  enabled: Scalars['Boolean']['output'];
  executions: Array<AdminCronJobExecution>;
  expression: Scalars['String']['output'];
  id: Scalars['Int64']['output'];
  lastExecAt?: Maybe<Scalars['Time']['output']>;
  name: Scalars['String']['output'];
};

export type AdminCronJobExecution = {
  __typename?: 'AdminCronJobExecution';
  errorMessage?: Maybe<Scalars['String']['output']>;
  finishedAt?: Maybe<Scalars['Time']['output']>;
  id: Scalars['Int64']['output'];
  job: AdminCronJob;
  jobId: Scalars['Int64']['output'];
  reportData?: Maybe<Scalars['String']['output']>;
  startedAt: Scalars['Time']['output'];
  status: CronJobExecutionStatus;
};

export type AdminCronJobsConnection = {
  __typename?: 'AdminCronJobsConnection';
  nodes: Array<AdminCronJob>;
};

export type AdminGitToken = {
  __typename?: 'AdminGitToken';
  canPull: Scalars['Boolean']['output'];
  canPush: Scalars['Boolean']['output'];
  createdAt: Scalars['Time']['output'];
  createdBy: AdminUser;
  description: Scalars['String']['output'];
  disabledAt?: Maybe<Scalars['Time']['output']>;
  disabledBy?: Maybe<AdminUser>;
  id: Scalars['Int64']['output'];
};

export type AdminGitTokensConnection = {
  __typename?: 'AdminGitTokensConnection';
  nodes: Array<AdminGitToken>;
};

export type AdminHtmlInjection = {
  __typename?: 'AdminHtmlInjection';
  activeFrom?: Maybe<Scalars['Time']['output']>;
  activeTo?: Maybe<Scalars['Time']['output']>;
  content: Scalars['String']['output'];
  createdAt: Scalars['Time']['output'];
  description: Scalars['String']['output'];
  id: Scalars['Int64']['output'];
  placement: Scalars['String']['output'];
  position: Scalars['Int']['output'];
};

export type AdminHtmlInjectionsConnection = {
  __typename?: 'AdminHtmlInjectionsConnection';
  nodes: Array<AdminHtmlInjection>;
};

export type AdminLatestNoteAssetsConnection = {
  __typename?: 'AdminLatestNoteAssetsConnection';
  nodes: Array<AdminNoteAsset>;
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
  createConfigVersion: CreateConfigVersionOrErrorPayload;
  createGitToken: CreateGitTokenOrErrorPayload;
  createHtmlInjection: CreateHtmlInjectionOrErrorPayload;
  createNotFoundIgnoredPattern: CreateNotFoundIgnoredPatternOrErrorPayload;
  createOffer: CreateOfferOrErrorPayload;
  createPatreonCredentials: CreatePatreonCredentialsOrErrorPayload;
  createRedirect: CreateRedirectOrErrorPayload;
  createRelease: CreateReleaseOrErrorPayload;
  createTgBot: CreateTgBotOrErrorPayload;
  createUser: CreateUserOrErrorPayload;
  deleteBoostyCredentials: DeleteBoostyCredentialsOrErrorPayload;
  deleteHtmlInjection: DeleteHtmlInjectionOrErrorPayload;
  deleteNotFoundIgnoredPattern: DeleteNotFoundIgnoredPatternOrErrorPayload;
  deletePatreonCredentials: DeletePatreonCredentialsOrErrorPayload;
  deleteRedirect: DeleteRedirectOrErrorPayload;
  disableApiKey: DisableApiKeyOrErrorPayload;
  disableGitToken: DisableGitTokenOrErrorPayload;
  makeReleaseLive: MakeReleaseLiveOrErrorPayload;
  refreshBoostyData: RefreshBoostyDataOrErrorPayload;
  refreshPatreonData: RefreshPatreonDataOrErrorPayload;
  removeExpiredTgChatMembers: RemoveExpiredTgChatMembersOrErrorPayload;
  resetNotFoundPath: ResetNotFoundPathOrErrorPayload;
  resetTelegramPublishNote: ResetTelegramPublishNoteOrErrorPayload;
  restoreBoostyCredentials: RestoreBoostyCredentialsOrErrorPayload;
  restorePatreonCredentials: RestorePatreonCredentialsOrErrorPayload;
  runCronJob: RunCronJobOrErrorPayload;
  setBoostyTierSubgraphs: SetBoostyTierSubgraphsOrErrorPayload;
  setPatreonTierSubgraphs: SetPatreonTierSubgraphsOrErrorPayload;
  setTgChatPublishInstantTags: SetTgChatPublishInstantTagsOrErrorPayload;
  setTgChatPublishTags: SetTgChatPublishTagsOrErrorPayload;
  setTgChatSubgraphInvites: SetTgChatSubgraphInvitesOrErrorPayload;
  setTgChatSubgraphs: SetTgChatSubgraphsOrErrorPayload;
  unbanUser: UnbanUserOrErrorPayload;
  updateBoostyCredentials: UpdateBoostyCredentialsOrErrorPayload;
  updateCronJob: UpdateCronJobOrErrorPayload;
  updateHtmlInjection: UpdateHtmlInjectionOrErrorPayload;
  updateNotFoundIgnoredPattern: UpdateNotFoundIgnoredPatternOrErrorPayload;
  updateNoteGraphPositions: UpdateNoteGraphPositionsOrErrorPayload;
  updateOffer: UpdateOfferOrErrorPayload;
  updateRedirect: UpdateRedirectOrErrorPayload;
  updateSubgraph: UpdateSubgraphOrErrorPayload;
  updateTgBot: UpdateTgBotOrErrorPayload;
  updateUser: UpdateUserOrErrorPayload;
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


export type AdminMutationCreateConfigVersionArgs = {
  input: CreateConfigVersionInput;
};


export type AdminMutationCreateGitTokenArgs = {
  input: CreateGitTokenInput;
};


export type AdminMutationCreateHtmlInjectionArgs = {
  input: CreateHtmlInjectionInput;
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


export type AdminMutationCreateUserArgs = {
  input: CreateUserInput;
};


export type AdminMutationDeleteBoostyCredentialsArgs = {
  input: DeleteBoostyCredentialsInput;
};


export type AdminMutationDeleteHtmlInjectionArgs = {
  input: DeleteHtmlInjectionInput;
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


export type AdminMutationDisableGitTokenArgs = {
  input: DisableGitTokenInput;
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


export type AdminMutationRemoveExpiredTgChatMembersArgs = {
  input: RemoveExpiredTgChatMembersInput;
};


export type AdminMutationResetNotFoundPathArgs = {
  input: ResetNotFoundPathInput;
};


export type AdminMutationResetTelegramPublishNoteArgs = {
  input: ResetTelegramPublishNoteInput;
};


export type AdminMutationRestoreBoostyCredentialsArgs = {
  input: RestoreBoostyCredentialsInput;
};


export type AdminMutationRestorePatreonCredentialsArgs = {
  input: RestorePatreonCredentialsInput;
};


export type AdminMutationRunCronJobArgs = {
  input: RunCronJobInput;
};


export type AdminMutationSetBoostyTierSubgraphsArgs = {
  input: SetBoostyTierSubgraphsInput;
};


export type AdminMutationSetPatreonTierSubgraphsArgs = {
  input: SetPatreonTierSubgraphsInput;
};


export type AdminMutationSetTgChatPublishInstantTagsArgs = {
  input: SetTgChatPublishInstantTagsInput;
};


export type AdminMutationSetTgChatPublishTagsArgs = {
  input: SetTgChatPublishTagsInput;
};


export type AdminMutationSetTgChatSubgraphInvitesArgs = {
  input: SetTgChatSubgraphInvitesInput;
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


export type AdminMutationUpdateCronJobArgs = {
  input: UpdateCronJobInput;
};


export type AdminMutationUpdateHtmlInjectionArgs = {
  input: UpdateHtmlInjectionInput;
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


export type AdminMutationUpdateUserArgs = {
  input: UpdateUserInput;
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

export type AdminNoteAsset = {
  __typename?: 'AdminNoteAsset';
  absolutePath: Scalars['String']['output'];
  createdAt: Scalars['Time']['output'];
  fileName: Scalars['String']['output'];
  id: Scalars['Int64']['output'];
  size: Scalars['Int64']['output'];
  url: Scalars['String']['output'];
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
  activeUserSubgraphs: Array<Scalars['String']['output']>;
  allAdmins: AdminAdminsConnection;
  allApiKeys: AdminApiKeysConnection;
  allBoostyCredentials: AdminBoostyCredentialsConnection;
  allConfigVersions: AdminConfigVersionsConnection;
  allCronJobs: AdminCronJobsConnection;
  allGitTokens: AdminGitTokensConnection;
  allHtmlInjections: AdminHtmlInjectionsConnection;
  allLatestNoteAssets: AdminLatestNoteAssetsConnection;
  allLatestNoteViews: AdminLatestNoteViewsConnection;
  allNotFoundIgnoredPatterns: AdminNotFoundIgnoredPatternsConnection;
  allNotFoundPaths: AdminNotFoundPathsConnection;
  allOffers: AdminOffersConnection;
  allPatreonCredentials: AdminPatreonCredentialsConnection;
  allPurchases: AdminPurchasesConnection;
  allRedirects: AdminRedirectsConnection;
  allReleases: AdminReleasesConnection;
  allSubgraphs: AdminSubgraphsConnection;
  allTelegramPublishNotes: AdminTelegramPublishNotesConnection;
  allTelegramPublishTags: AdminTelegramPublishTagsConnection;
  allTgBots: AdminTgBotsConnection;
  allUserSubgraphAccesses: AdminUserSubgraphAccessesConnection;
  allUserUserBans: AdminUserBansConnection;
  allUsers: AdminUsersConnection;
  allWaitListEmailRequests: AdminWaitListEmailRequestsConnection;
  allWaitListTgBotRequests: AdminWaitListTgBotRequestsConnection;
  apiKeyLogs: AdminApiKeyLogsConnection;
  auditLogs: AdminAuditLogsConnection;
  boostyCredentials?: Maybe<AdminBoostyCredentials>;
  cronJob?: Maybe<AdminCronJob>;
  htmlInjection?: Maybe<AdminHtmlInjection>;
  latestConfig: AdminConfigVersion;
  noteAsset?: Maybe<AdminNoteAsset>;
  noteView?: Maybe<NoteView>;
  offer?: Maybe<AdminOffer>;
  patreonCredentials?: Maybe<AdminPatreonCredentials>;
  purchase?: Maybe<AdminPurchase>;
  redirect?: Maybe<AdminRedirect>;
  subgraph?: Maybe<AdminSubgraph>;
  telegramPublishNote?: Maybe<AdminTelegramPublishNote>;
  tgBot?: Maybe<AdminTgBot>;
  tgBotChats: AdminTgBotChatsConnection;
  tgChatMembers: AdminTgChatMembersConnection;
  tgChatSubgraphAccesses: AdminTgChatSubgraphAccessesConnection;
  user?: Maybe<AdminUser>;
  userSubgraphAccess?: Maybe<AdminUserSubgraphAccess>;
};


export type AdminQueryActiveUserSubgraphsArgs = {
  id: Scalars['Int64']['input'];
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


export type AdminQueryAllTelegramPublishNotesArgs = {
  filter?: InputMaybe<AdminTelegramPublishNotesFilter>;
};


export type AdminQueryApiKeyLogsArgs = {
  filter: ApiKeyLogsFilterInput;
};


export type AdminQueryAuditLogsArgs = {
  filter: AdminAuditLogsFilterInput;
};


export type AdminQueryBoostyCredentialsArgs = {
  id: Scalars['Int64']['input'];
};


export type AdminQueryCronJobArgs = {
  id: Scalars['Int64']['input'];
};


export type AdminQueryHtmlInjectionArgs = {
  id: Scalars['Int64']['input'];
};


export type AdminQueryNoteAssetArgs = {
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


export type AdminQueryTelegramPublishNoteArgs = {
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


export type AdminQueryUserArgs = {
  id: Scalars['Int64']['input'];
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

export type AdminTelegramPublishNote = {
  __typename?: 'AdminTelegramPublishNote';
  chats: Array<AdminTgBotChat>;
  createdAt: Scalars['Time']['output'];
  id: Scalars['Int64']['output'];
  /** latest or published NoteView depending on the status */
  noteView: NoteView;
  post: TelegramPost;
  publishAt: Scalars['Time']['output'];
  publishedAt?: Maybe<Scalars['Time']['output']>;
  publishedVersionID?: Maybe<Scalars['Int64']['output']>;
  secondsUntilPublish: Scalars['Int64']['output'];
  status: Scalars['String']['output'];
  tags: Array<AdminTelegramPublishTag>;
};

export type AdminTelegramPublishNotesConnection = {
  __typename?: 'AdminTelegramPublishNotesConnection';
  count: Scalars['Int64']['output'];
  nodes: Array<AdminTelegramPublishNote>;
};

export type AdminTelegramPublishNotesFilter = {
  includeOutdated?: InputMaybe<Scalars['Boolean']['input']>;
  includeSent?: InputMaybe<Scalars['Boolean']['input']>;
};

export type AdminTelegramPublishTag = {
  __typename?: 'AdminTelegramPublishTag';
  createdAt: Scalars['Time']['output'];
  id: Scalars['Int64']['output'];
  label: Scalars['String']['output'];
};

export type AdminTelegramPublishTagsConnection = {
  __typename?: 'AdminTelegramPublishTagsConnection';
  nodes: Array<AdminTelegramPublishTag>;
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
  canInvite: Scalars['Boolean']['output'];
  chatTitle: Scalars['String']['output'];
  chatType: Scalars['String']['output'];
  id: Scalars['Int64']['output'];
  memberCount: Scalars['Int']['output'];
  publishInstantTags: Array<AdminTelegramPublishTag>;
  publishTags: Array<AdminTelegramPublishTag>;
  removedAt?: Maybe<Scalars['Time']['output']>;
  subgraphAccesses: Array<AdminTgChatSubgraphAccess>;
  subgraphInvites: Array<AdminTgBotChatSubgraphInvite>;
};

export type AdminTgBotChatSubgraphInvite = {
  __typename?: 'AdminTgBotChatSubgraphInvite';
  chat: AdminTgBotChat;
  chatId: Scalars['Int64']['output'];
  createdAt: Scalars['Time']['output'];
  id: Scalars['String']['output'];
  subgraph: AdminSubgraph;
  subgraphId: Scalars['Int64']['output'];
};

export type AdminTgBotChatsConnection = {
  __typename?: 'AdminTgBotChatsConnection';
  nodes: Array<AdminTgBotChat>;
};

export type AdminTgBotChatsFilterInput = {
  botId?: InputMaybe<Scalars['Int64']['input']>;
  canInvite?: InputMaybe<Scalars['Boolean']['input']>;
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

export type AdminWaitListEmailRequest = {
  __typename?: 'AdminWaitListEmailRequest';
  createdAt: Scalars['Time']['output'];
  email: Scalars['String']['output'];
  ip?: Maybe<Scalars['String']['output']>;
  notePath: Scalars['String']['output'];
};

export type AdminWaitListEmailRequestsConnection = {
  __typename?: 'AdminWaitListEmailRequestsConnection';
  nodes: Array<AdminWaitListEmailRequest>;
};

export type AdminWaitListTgBotRequest = {
  __typename?: 'AdminWaitListTgBotRequest';
  botName: Scalars['String']['output'];
  chatId: Scalars['Int64']['output'];
  createdAt: Scalars['Time']['output'];
  notePath: Scalars['String']['output'];
  notePathId: Scalars['Int64']['output'];
};

export type AdminWaitListTgBotRequestsConnection = {
  __typename?: 'AdminWaitListTgBotRequestsConnection';
  nodes: Array<AdminWaitListTgBotRequest>;
};

export type ApiKeyLogsFilterInput = {
  apiKeyId?: InputMaybe<Scalars['Int64']['input']>;
};

export enum AuditLogLevelEnum {
  Debug = 'DEBUG',
  Error = 'ERROR',
  Info = 'INFO',
  Unknown = 'UNKNOWN',
  Warning = 'WARNING'
}

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

export type CreateConfigVersionInput = {
  defaultLayout: Scalars['String']['input'];
  showDraftVersions: Scalars['Boolean']['input'];
  timezone: Scalars['String']['input'];
};

export type CreateConfigVersionOrErrorPayload = CreateConfigVersionPayload | ErrorPayload;

export type CreateConfigVersionPayload = {
  __typename?: 'CreateConfigVersionPayload';
  configVersion: AdminConfigVersion;
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

export type CreateGitTokenInput = {
  canPull: Scalars['Boolean']['input'];
  canPush: Scalars['Boolean']['input'];
  description: Scalars['String']['input'];
};

export type CreateGitTokenOrErrorPayload = CreateGitTokenPayload | ErrorPayload;

export type CreateGitTokenPayload = {
  __typename?: 'CreateGitTokenPayload';
  gitToken: AdminGitToken;
  value: Scalars['String']['output'];
};

export type CreateHtmlInjectionInput = {
  activeFrom?: InputMaybe<Scalars['Time']['input']>;
  activeTo?: InputMaybe<Scalars['Time']['input']>;
  content: Scalars['String']['input'];
  description: Scalars['String']['input'];
  placement: Scalars['String']['input'];
  position: Scalars['Int']['input'];
};

export type CreateHtmlInjectionOrErrorPayload = CreateHtmlInjectionPayload | ErrorPayload;

export type CreateHtmlInjectionPayload = {
  __typename?: 'CreateHtmlInjectionPayload';
  htmlInjection: AdminHtmlInjection;
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

export type CreateUserInput = {
  email: Scalars['String']['input'];
};

export type CreateUserOrErrorPayload = CreateUserPayload | ErrorPayload;

export type CreateUserPayload = {
  __typename?: 'CreateUserPayload';
  user: AdminUser;
};

export enum CronJobExecutionStatus {
  Completed = 'COMPLETED',
  Failed = 'FAILED',
  Pending = 'PENDING',
  Running = 'RUNNING'
}

export type DeleteBoostyCredentialsInput = {
  id: Scalars['Int64']['input'];
};

export type DeleteBoostyCredentialsOrErrorPayload = DeleteBoostyCredentialsPayload | ErrorPayload;

export type DeleteBoostyCredentialsPayload = {
  __typename?: 'DeleteBoostyCredentialsPayload';
  boostyCredentials: AdminBoostyCredentials;
  deletedId: Scalars['Int64']['output'];
};

export type DeleteHtmlInjectionInput = {
  id: Scalars['Int64']['input'];
};

export type DeleteHtmlInjectionOrErrorPayload = DeleteHtmlInjectionPayload | ErrorPayload;

export type DeleteHtmlInjectionPayload = {
  __typename?: 'DeleteHtmlInjectionPayload';
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

export type DisableGitTokenInput = {
  id: Scalars['Int64']['input'];
};

export type DisableGitTokenOrErrorPayload = DisableGitTokenPayload | ErrorPayload;

export type DisableGitTokenPayload = {
  __typename?: 'DisableGitTokenPayload';
  gitToken: AdminGitToken;
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

export type GenerateTgAttachCodeInput = {
  botId: Scalars['Int64']['input'];
};

export type GenerateTgAttachCodeOrErrorPayload = ErrorPayload | GenerateTgAttachCodePayload;

export type GenerateTgAttachCodePayload = {
  __typename?: 'GenerateTgAttachCodePayload';
  code: Scalars['String']['output'];
  url: Scalars['String']['output'];
};

export type HideNotesInput = {
  paths: Array<Scalars['String']['input']>;
};

export type HideNotesOrErrorPayload = ErrorPayload | HideNotesPayload;

export type HideNotesPayload = {
  __typename?: 'HideNotesPayload';
  success: Scalars['Boolean']['output'];
};

export type LastNoteReadAtInput = {
  pathId: Scalars['Int64']['input'];
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
  generateTgAttachCode: GenerateTgAttachCodeOrErrorPayload;
  /** X-Api-Key header must be set. */
  hideNotes: HideNotesOrErrorPayload;
  /** X-Api-Key header must be set. */
  pushNotes: PushNotesOrErrorPayload;
  requestEmailSignInCode: RequestEmailSignInCodeOrErrorPayload;
  signInByEmail: SignInOrErrorPayload;
  signOut: SignOutOrErrorPayload;
  toggleFavoriteNote: ToggleFavoriteNoteOrErrorPayload;
  /** X-Api-Key header must be set. */
  uploadNoteAsset: UploadNoteAssetOrErrorPayload;
};


export type MutationCreateEmailWaitListRequestArgs = {
  input: CreateEmailWaitListRequestInput;
};


export type MutationCreatePaymentLinkArgs = {
  input: CreatePaymentLinkInput;
};


export type MutationGenerateTgAttachCodeArgs = {
  input: GenerateTgAttachCodeInput;
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


export type MutationToggleFavoriteNoteArgs = {
  input: ToggleFavoriteNoteInput;
};


export type MutationUploadNoteAssetArgs = {
  input: UploadNoteAssetInput;
};

export type NoteInput = {
  path?: InputMaybe<Scalars['String']['input']>;
  pathId?: InputMaybe<Scalars['Int64']['input']>;
  referer: Scalars['String']['input'];
};

export type NotePath = {
  __typename?: 'NotePath';
  latestContentHash: Scalars['String']['output'];
  latestNoteView: NoteView;
  value: Scalars['String']['output'];
};

export type NotePathsFilter = {
  /**
   * LIKE pattern with % and _ wildcards supported.
   * For example, to find all note paths starting with "myfolder/", use "myfolder/%".
   */
  like?: InputMaybe<Scalars['String']['input']>;
  /** Full-text search on note paths. like will be ignored if search is set. */
  search?: InputMaybe<Scalars['String']['input']>;
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
  description?: Maybe<Scalars['String']['output']>;
  free: Scalars['Boolean']['output'];
  graphPosition?: Maybe<Vector2>;
  html: Scalars['String']['output'];
  id: Scalars['String']['output'];
  inLinks: Array<NoteView>;
  isHomePage: Scalars['Boolean']['output'];
  meta: Array<NoteViewMeta>;
  path: Scalars['String']['output'];
  pathId: Scalars['Int64']['output'];
  permalink: Scalars['String']['output'];
  subgraphNames: Array<Scalars['String']['output']>;
  title: Scalars['String']['output'];
  toc: Array<NoteTocItem>;
  versionId: Scalars['Int64']['output'];
  warnings: Array<NoteWarning>;
};

export type NoteViewMeta = {
  __typename?: 'NoteViewMeta';
  key: Scalars['String']['output'];
  raw: Scalars['String']['output'];
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
  pathId: Scalars['Int64']['output'];
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
  /** X-Api-Key header must be set. */
  notePaths: Array<NotePath>;
  search: SearchConnection;
  viewer: Viewer;
};


export type QueryNoteArgs = {
  input: NoteInput;
};


export type QueryNotePathsArgs = {
  filter?: InputMaybe<NotePathsFilter>;
};


export type QuerySearchArgs = {
  input: SearchInput;
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

export type RemoveExpiredTgChatMembersInput = {
  chatId?: InputMaybe<Scalars['Int64']['input']>;
  userId?: InputMaybe<Scalars['Int64']['input']>;
};

export type RemoveExpiredTgChatMembersOrErrorPayload = ErrorPayload | RemoveExpiredTgChatMembersPayload;

export type RemoveExpiredTgChatMembersPayload = {
  __typename?: 'RemoveExpiredTgChatMembersPayload';
  errors: Array<Scalars['String']['output']>;
  removedCount: Scalars['Int']['output'];
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

export type ResetTelegramPublishNoteInput = {
  id: Scalars['Int64']['input'];
};

export type ResetTelegramPublishNoteOrErrorPayload = ErrorPayload | ResetTelegramPublishNotePayload;

export type ResetTelegramPublishNotePayload = {
  __typename?: 'ResetTelegramPublishNotePayload';
  publishNote: AdminTelegramPublishNote;
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

export type RunCronJobInput = {
  id: Scalars['Int64']['input'];
};

export type RunCronJobOrErrorPayload = ErrorPayload | RunCronJobPayload;

export type RunCronJobPayload = {
  __typename?: 'RunCronJobPayload';
  execution: AdminCronJobExecution;
};

export type SearchConnection = {
  __typename?: 'SearchConnection';
  nodes: Array<SearchResult>;
  totalCount: Scalars['Int64']['output'];
};

export type SearchInput = {
  query: Scalars['String']['input'];
};

export type SearchResult = {
  __typename?: 'SearchResult';
  document?: Maybe<SearchResultDocument>;
  highlightedContent: Array<Scalars['String']['output']>;
  highlightedTitle?: Maybe<Scalars['String']['output']>;
  url: Scalars['String']['output'];
};

export type SearchResultDocument = PublicNote;

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

export type SetTgChatPublishInstantTagsInput = {
  chatId: Scalars['Int64']['input'];
  tagIds: Array<Scalars['Int64']['input']>;
};

export type SetTgChatPublishInstantTagsOrErrorPayload = ErrorPayload | SetTgChatPublishInstantTagsPayload;

export type SetTgChatPublishInstantTagsPayload = {
  __typename?: 'SetTgChatPublishInstantTagsPayload';
  chat: AdminTgBotChat;
  success: Scalars['Boolean']['output'];
};

export type SetTgChatPublishTagsInput = {
  chatId: Scalars['Int64']['input'];
  tagIds: Array<Scalars['Int64']['input']>;
};

export type SetTgChatPublishTagsOrErrorPayload = ErrorPayload | SetTgChatPublishTagsPayload;

export type SetTgChatPublishTagsPayload = {
  __typename?: 'SetTgChatPublishTagsPayload';
  chat: AdminTgBotChat;
  success: Scalars['Boolean']['output'];
};

export type SetTgChatSubgraphInvitesInput = {
  chatId: Scalars['Int64']['input'];
  subgraphIds: Array<Scalars['Int64']['input']>;
};

export type SetTgChatSubgraphInvitesOrErrorPayload = ErrorPayload | SetTgChatSubgraphInvitesPayload;

export type SetTgChatSubgraphInvitesPayload = {
  __typename?: 'SetTgChatSubgraphInvitesPayload';
  chat: AdminTgBotChat;
  success: Scalars['Boolean']['output'];
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

export type TelegramPost = {
  __typename?: 'TelegramPost';
  content: Scalars['String']['output'];
  warnings: Array<Scalars['String']['output']>;
};

export type TgBot = {
  __typename?: 'TgBot';
  description: Scalars['String']['output'];
  id: Scalars['Int64']['output'];
};

export type ToggleFavoriteNoteInput = {
  pathId: Scalars['Int64']['input'];
  value: Scalars['Boolean']['input'];
};

export type ToggleFavoriteNoteOrErrorPayload = ErrorPayload | ToggleFavoriteNotePayload;

export type ToggleFavoriteNotePayload = {
  __typename?: 'ToggleFavoriteNotePayload';
  favoriteNotes: Array<PublicNote>;
  success: Scalars['Boolean']['output'];
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

export type UpdateCronJobInput = {
  enabled: Scalars['Boolean']['input'];
  expression: Scalars['String']['input'];
  id: Scalars['Int64']['input'];
};

export type UpdateCronJobOrErrorPayload = ErrorPayload | UpdateCronJobPayload;

export type UpdateCronJobPayload = {
  __typename?: 'UpdateCronJobPayload';
  cronJob: AdminCronJob;
};

export type UpdateHtmlInjectionInput = {
  activeFrom?: InputMaybe<Scalars['Time']['input']>;
  activeTo?: InputMaybe<Scalars['Time']['input']>;
  content: Scalars['String']['input'];
  description: Scalars['String']['input'];
  id: Scalars['Int64']['input'];
  placement: Scalars['String']['input'];
  position: Scalars['Int']['input'];
};

export type UpdateHtmlInjectionOrErrorPayload = ErrorPayload | UpdateHtmlInjectionPayload;

export type UpdateHtmlInjectionPayload = {
  __typename?: 'UpdateHtmlInjectionPayload';
  htmlInjection: AdminHtmlInjection;
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

export type UpdateUserInput = {
  email?: InputMaybe<Scalars['String']['input']>;
  id: Scalars['Int64']['input'];
};

export type UpdateUserOrErrorPayload = ErrorPayload | UpdateUserPayload;

export type UpdateUserPayload = {
  __typename?: 'UpdateUserPayload';
  user: AdminUser;
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
  favoriteNotes: Array<PublicNote>;
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
  lastNoteReadAt?: Maybe<Scalars['Time']['output']>;
  offers?: Maybe<ViewerOffers>;
  role: Role;
  tgBots: Array<TgBot>;
  user?: Maybe<User>;
};


export type ViewerLastNoteReadAtArgs = {
  input: LastNoteReadAtInput;
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

export type AdminAuditLogsQueryVariables = Exact<{
  filter: AdminAuditLogsFilterInput;
}>;


export type AdminAuditLogsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', auditLogs: { __typename?: 'AdminAuditLogsConnection', nodes: Array<{ __typename?: 'AdminAuditLog', id: any, createdAt: any, level: AuditLogLevelEnum, message: string, params: string }> } } };

export type AdminDeleteBoostyCredentialsMutationVariables = Exact<{
  input: DeleteBoostyCredentialsInput;
}>;


export type AdminDeleteBoostyCredentialsMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'DeleteBoostyCredentialsPayload', boostyCredentials: { __typename?: 'AdminBoostyCredentials', id: any } } | { __typename?: 'ErrorPayload', message: string } } };

export type RefreshBoostyDataMutationVariables = Exact<{
  input: RefreshBoostyDataInput;
}>;


export type RefreshBoostyDataMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'RefreshBoostyDataPayload', success: boolean, credentials: { __typename?: 'AdminBoostyCredentials', id: any } } } };

export type AdminRestoreBoostyCredentialsMutationVariables = Exact<{
  input: RestoreBoostyCredentialsInput;
}>;


export type AdminRestoreBoostyCredentialsMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'RestoreBoostyCredentialsPayload', boostyCredentials: { __typename?: 'AdminBoostyCredentials', id: any } } } };

export type AdminBoostyCredentialsQueryVariables = Exact<{
  filter?: InputMaybe<AdminBoostyCredentialsFilterInput>;
}>;


export type AdminBoostyCredentialsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allBoostyCredentials: { __typename?: 'AdminBoostyCredentialsConnection', nodes: Array<{ __typename?: 'AdminBoostyCredentials', id: any, state: BoostyCredentialsStateEnum, deviceId: string, blogName: string, createdAt: any, createdBy: { __typename?: 'AdminUser', id: any, email?: string | null } }> } } };

export type AdminCreateBoostyCredsMutationVariables = Exact<{
  input: CreateBoostyCredentialsInput;
}>;


export type AdminCreateBoostyCredsMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'CreateBoostyCredentialsPayload', boostyCredentials: { __typename?: 'AdminBoostyCredentials', id: any } } | { __typename?: 'ErrorPayload', message: string } } };

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

export type AdminConfigVersionsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminConfigVersionsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allConfigVersions: { __typename?: 'AdminConfigVersionsConnection', nodes: Array<{ __typename?: 'AdminConfigVersion', id: any, createdAt: any, showDraftVersions: boolean, defaultLayout: string, timezone: string, createdBy: { __typename?: 'AdminUser', email?: string | null } }> } } };

export type AdminCreateConfigLatestConfigQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminCreateConfigLatestConfigQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', latestConfig: { __typename?: 'AdminConfigVersion', showDraftVersions: boolean, defaultLayout: string, timezone: string } } };

export type AdminCreateConfigVersionMutationVariables = Exact<{
  input: CreateConfigVersionInput;
}>;


export type AdminCreateConfigVersionMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'CreateConfigVersionPayload', configVersion: { __typename?: 'AdminConfigVersion', id: any } } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminRunCronJobMutationVariables = Exact<{
  input: RunCronJobInput;
}>;


export type AdminRunCronJobMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', runCronJob: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'RunCronJobPayload', execution: { __typename?: 'AdminCronJobExecution', id: any, job: { __typename?: 'AdminCronJob', id: any } } } } };

export type AdminAllCronJobsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminAllCronJobsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allCronJobs: { __typename?: 'AdminCronJobsConnection', nodes: Array<{ __typename?: 'AdminCronJob', id: any, name: string, enabled: boolean, expression: string, lastExecAt?: any | null }> } } };

export type AdminCronJobExecutionsQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminCronJobExecutionsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', cronJob?: { __typename?: 'AdminCronJob', id: any, executions: Array<{ __typename?: 'AdminCronJobExecution', id: any, startedAt: any, finishedAt?: any | null, status: CronJobExecutionStatus, errorMessage?: string | null }> } | null } };

export type AdminCronJobShowQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminCronJobShowQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', cronJob?: { __typename?: 'AdminCronJob', id: any, name: string, enabled: boolean, expression: string, lastExecAt?: any | null } | null } };

export type AdminCronJobUpdateQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminCronJobUpdateQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', cronJob?: { __typename?: 'AdminCronJob', id: any, name: string, enabled: boolean, expression: string, lastExecAt?: any | null } | null } };

export type AdminUpdateCronJobMutationVariables = Exact<{
  input: UpdateCronJobInput;
}>;


export type AdminUpdateCronJobMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', updateCronJob: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'UpdateCronJobPayload', cronJob: { __typename?: 'AdminCronJob', id: any, expression: string, enabled: boolean } } } };

export type DisableGitTokenMutationVariables = Exact<{
  input: DisableGitTokenInput;
}>;


export type DisableGitTokenMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'DisableGitTokenPayload', gitToken: { __typename?: 'AdminGitToken', id: any } } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminGitTokensQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminGitTokensQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allGitTokens: { __typename?: 'AdminGitTokensConnection', nodes: Array<{ __typename?: 'AdminGitToken', id: any, createdAt: any, description: string, canPull: boolean, canPush: boolean, disabledAt?: any | null, createdBy: { __typename?: 'AdminUser', id: any, email?: string | null }, disabledBy?: { __typename?: 'AdminUser', id: any, email?: string | null } | null }> } } };

export type AdminCreateGitTokenMutationVariables = Exact<{
  input: CreateGitTokenInput;
}>;


export type AdminCreateGitTokenMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'CreateGitTokenPayload', value: string, gitToken: { __typename?: 'AdminGitToken', id: any } } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminDeleteHtmlInjectionMutationVariables = Exact<{
  input: DeleteHtmlInjectionInput;
}>;


export type AdminDeleteHtmlInjectionMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename: 'DeleteHtmlInjectionPayload', deletedId: any } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminHtmlInjectionsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminHtmlInjectionsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allHtmlInjections: { __typename?: 'AdminHtmlInjectionsConnection', nodes: Array<{ __typename?: 'AdminHtmlInjection', id: any, createdAt: any, activeFrom?: any | null, activeTo?: any | null, description: string, position: number, placement: string }> } } };

export type AdminCreateHtmlInjectionMutationMutationVariables = Exact<{
  input: CreateHtmlInjectionInput;
}>;


export type AdminCreateHtmlInjectionMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'CreateHtmlInjectionPayload', htmlInjection: { __typename?: 'AdminHtmlInjection', id: any } } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminShowHtmlInjectionQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminShowHtmlInjectionQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', htmlInjection?: { __typename?: 'AdminHtmlInjection', id: any, createdAt: any, activeFrom?: any | null, activeTo?: any | null, description: string, position: number, placement: string, content: string } | null } };

export type AdminUpdateDataHtmlInjectionQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminUpdateDataHtmlInjectionQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', htmlInjection?: { __typename?: 'AdminHtmlInjection', id: any, createdAt: any, activeFrom?: any | null, activeTo?: any | null, description: string, position: number, placement: string, content: string } | null } };

export type AdminUpdateHtmlInjectionMutationVariables = Exact<{
  input: UpdateHtmlInjectionInput;
}>;


export type AdminUpdateHtmlInjectionMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'UpdateHtmlInjectionPayload', htmlInjection: { __typename?: 'AdminHtmlInjection', id: any } } } };

export type AdminNoteAssetsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminNoteAssetsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allLatestNoteAssets: { __typename?: 'AdminLatestNoteAssetsConnection', nodes: Array<{ __typename?: 'AdminNoteAsset', id: any, absolutePath: string, fileName: string, size: any }> } } };

export type AdminNoteAssetQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminNoteAssetQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', noteAsset?: { __typename?: 'AdminNoteAsset', id: any, absolutePath: string, fileName: string, size: any, createdAt: any, url: string } | null } };

export type AdminListNoteViewsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminListNoteViewsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allLatestNoteViews: { __typename?: 'AdminLatestNoteViewsConnection', nodes: Array<{ __typename?: 'NoteView', id: string, path: string, title: string, free: boolean }> } } };

export type AdminGraphQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminGraphQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename?: 'AdminSubgraph', name: string, color?: string | null }> }, allLatestNoteViews: { __typename?: 'AdminLatestNoteViewsConnection', nodes: Array<{ __typename?: 'NoteView', id: string, subgraphNames: Array<string>, title: string, pathId: any, free: boolean, isHomePage: boolean, graphPosition?: { __typename?: 'Vector2', x: number, y: number } | null, inLinks: Array<{ __typename?: 'NoteView', title: string, pathId: any, id: string }> }> } } };

export type UpdateNoteGraphPositionsMutationVariables = Exact<{
  input: UpdateNoteGraphPositionsInput;
}>;


export type UpdateNoteGraphPositionsMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'UpdateNoteGraphPositionsPayload', success: boolean } } };

export type AdminSelectNoteViewQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminSelectNoteViewQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allLatestNoteViews: { __typename?: 'AdminLatestNoteViewsConnection', nodes: Array<{ __typename?: 'NoteView', versionId: any, path: string, title: string }> } } };

export type AdminNoteViewQueryVariables = Exact<{
  id: Scalars['String']['input'];
}>;


export type AdminNoteViewQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', noteView?: { __typename: 'NoteView', path: string, title: string, permalink: string } | null } };

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


export type AdminDeleteNotFoundIgnoredPatternMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'DeleteNotFoundIgnoredPatternPayload', deletedId: any } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminNotFoundIgnoredPatternsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminNotFoundIgnoredPatternsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allNotFoundIgnoredPatterns: { __typename?: 'AdminNotFoundIgnoredPatternsConnection', nodes: Array<{ __typename?: 'AdminNotFoundIgnoredPattern', id: any, pattern: string, createdAt: any, createdBy: { __typename?: 'AdminUser', id: any, email?: string | null } }> } } };

export type AdminCreateNotFoundIgnoredPatternMutationMutationVariables = Exact<{
  input: CreateNotFoundIgnoredPatternInput;
}>;


export type AdminCreateNotFoundIgnoredPatternMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'CreateNotFoundIgnoredPatternPayload', notFoundIgnoredPattern: { __typename?: 'AdminNotFoundIgnoredPattern', id: any } } | { __typename?: 'ErrorPayload', message: string } } };

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


export type AdminCreateOfferMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'CreateOfferPayload', offer: { __typename?: 'AdminOffer', id: any, publicId: string } } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminShowOfferQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminShowOfferQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', offer?: { __typename?: 'AdminOffer', id: any, publicId: string, createdAt: any, lifetime?: string | null, priceUSD: number, startsAt?: any | null, endsAt?: any | null, subgraphIds: Array<any>, subgraphs: Array<{ __typename?: 'AdminSubgraph', id: any, name: string }> } | null } };

export type AdminUpdateOfferMutationMutationVariables = Exact<{
  input: UpdateOfferInput;
}>;


export type AdminUpdateOfferMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'UpdateOfferPayload', offer: { __typename?: 'AdminOffer', id: any, publicId: string } } } };

export type AdminDeletePatreonCredentialsMutationVariables = Exact<{
  input: DeletePatreonCredentialsInput;
}>;


export type AdminDeletePatreonCredentialsMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'DeletePatreonCredentialsPayload', patreonCredentials: { __typename?: 'AdminPatreonCredentials', id: any } } | { __typename?: 'ErrorPayload', message: string } } };

export type RefreshPatreonDataMutationVariables = Exact<{
  input: RefreshPatreonDataInput;
}>;


export type RefreshPatreonDataMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'RefreshPatreonDataPayload', success: boolean } } };

export type AdminRestorePatreonCredentialsMutationVariables = Exact<{
  input: RestorePatreonCredentialsInput;
}>;


export type AdminRestorePatreonCredentialsMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'RestorePatreonCredentialsPayload', patreonCredentials: { __typename?: 'AdminPatreonCredentials', id: any } } } };

export type AdminPatreonCredentialsQueryVariables = Exact<{
  filter?: InputMaybe<AdminPatreonCredentialsFilterInput>;
}>;


export type AdminPatreonCredentialsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allPatreonCredentials: { __typename?: 'AdminPatreonCredentialsConnection', nodes: Array<{ __typename?: 'AdminPatreonCredentials', id: any, state: PatreonCredentialsStateEnum, creatorAccessToken: string, createdAt: any, syncedAt?: any | null, createdBy: { __typename?: 'AdminUser', id: any, email?: string | null } }> } } };

export type AdminCreatePatreonCredsMutationVariables = Exact<{
  input: CreatePatreonCredentialsInput;
}>;


export type AdminCreatePatreonCredsMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'CreatePatreonCredentialsPayload', patreonCredentials: { __typename?: 'AdminPatreonCredentials', id: any } } | { __typename?: 'ErrorPayload', message: string } } };

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


export type AdminCreateRedirectMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'CreateRedirectPayload', redirect: { __typename?: 'AdminRedirect', id: any } } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminShowRedirectQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminShowRedirectQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', redirect?: { __typename?: 'AdminRedirect', id: any, createdAt: any, pattern: string, ignoreCase: boolean, isRegex: boolean, target: string, createdBy: { __typename?: 'AdminUser', id: any, email?: string | null } } | null } };

export type AdminDeleteRedirectMutationMutationVariables = Exact<{
  input: DeleteRedirectInput;
}>;


export type AdminDeleteRedirectMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'DeleteRedirectPayload', id: any } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminUpdateRedirectMutationMutationVariables = Exact<{
  input: UpdateRedirectInput;
}>;


export type AdminUpdateRedirectMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'UpdateRedirectPayload', redirect: { __typename?: 'AdminRedirect', id: any } } } };

export type AdminMakeReleaseLiveMutationVariables = Exact<{
  input: MakeReleaseLiveInput;
}>;


export type AdminMakeReleaseLiveMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'MakeReleaseLivePayload', release: { __typename?: 'AdminRelease', id: any } } } };

export type AdminReleasesQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminReleasesQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allReleases: { __typename?: 'AdminReleasesConnection', nodes: Array<{ __typename?: 'AdminRelease', id: any, createdAt: any, title: string, isLive: boolean, createdBy: { __typename?: 'AdminUser', email?: string | null } }> } } };

export type AdminCreateReleaseMutationVariables = Exact<{
  input: CreateReleaseInput;
}>;


export type AdminCreateReleaseMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'CreateReleasePayload', release: { __typename?: 'AdminRelease', id: any } } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminListSubgraphsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminListSubgraphsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename: 'AdminSubgraph', id: any, name: string, color?: string | null, createdAt: any }> } } };

export type AdminSelectSubgraphListQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminSelectSubgraphListQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename?: 'AdminSubgraph', id: any, name: string }> } } };

export type AdminSelectSubgraphQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminSelectSubgraphQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename?: 'AdminSubgraph', id: any, name: string }> } } };

export type AdminShowSubgraphQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminShowSubgraphQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', subgraph?: { __typename?: 'AdminSubgraph', id: any, name: string, color?: string | null, hidden: boolean } | null } };

export type UpdateSubgraphMutationVariables = Exact<{
  input: UpdateSubgraphInput;
}>;


export type UpdateSubgraphMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'ErrorPayload', message: string } | { __typename: 'UpdateSubgraphPayload', subgraph: { __typename: 'AdminSubgraph', id: any, color?: string | null } } } };

export type AdminResetTelegramPublishNoteMutationVariables = Exact<{
  input: ResetTelegramPublishNoteInput;
}>;


export type AdminResetTelegramPublishNoteMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'ResetTelegramPublishNotePayload', publishNote: { __typename?: 'AdminTelegramPublishNote', id: any } } } };

export type AdminTelegramPublishNoteCountQueryVariables = Exact<{
  filter: AdminTelegramPublishNotesFilter;
}>;


export type AdminTelegramPublishNoteCountQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allTelegramPublishNotes: { __typename?: 'AdminTelegramPublishNotesConnection', count: any } } };

export type AdminTelegramPublishNotesQueryVariables = Exact<{
  filter: AdminTelegramPublishNotesFilter;
}>;


export type AdminTelegramPublishNotesQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allTelegramPublishNotes: { __typename?: 'AdminTelegramPublishNotesConnection', nodes: Array<{ __typename?: 'AdminTelegramPublishNote', id: any, publishAt: any, secondsUntilPublish: any, publishedAt?: any | null, status: string, noteView: { __typename?: 'NoteView', title: string } }> } } };

export type AdminTelegramPublishNoteQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminTelegramPublishNoteQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', telegramPublishNote?: { __typename?: 'AdminTelegramPublishNote', id: any, createdAt: any, publishAt: any, secondsUntilPublish: any, publishedAt?: any | null, status: string, tags: Array<{ __typename?: 'AdminTelegramPublishTag', label: string }>, chats: Array<{ __typename?: 'AdminTgBotChat', chatTitle: string, chatType: string }>, noteView: { __typename?: 'NoteView', title: string }, post: { __typename?: 'TelegramPost', content: string, warnings: Array<string> } } | null } };

export type AdminTgBotsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminTgBotsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allTgBots: { __typename?: 'AdminTgBotsConnection', nodes: Array<{ __typename?: 'AdminTgBot', id: any, name: string, description: string, enabled: boolean, createdAt: any, createdBy: { __typename?: 'AdminUser', email?: string | null } }> } } };

export type AdminCreateTgBotMutationMutationVariables = Exact<{
  input: CreateTgBotInput;
}>;


export type AdminCreateTgBotMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'CreateTgBotPayload', tgBot: { __typename?: 'AdminTgBot', id: any, name: string } } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminTgBotChatsQueryVariables = Exact<{
  filter: AdminTgBotChatsFilterInput;
}>;


export type AdminTgBotChatsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', tgBotChats: { __typename?: 'AdminTgBotChatsConnection', nodes: Array<{ __typename?: 'AdminTgBotChat', id: any, chatType: string, chatTitle: string, addedAt: any, removedAt?: any | null, memberCount: number, subgraphAccesses: Array<{ __typename?: 'AdminTgChatSubgraphAccess', id: any, subgraphId: any }> }> } } };

export type AdminTgbotShowChatsSubgraphsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminTgbotShowChatsSubgraphsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename?: 'AdminSubgraph', id: any, name: string }> } } };

export type AdminTgbotsShowchatsSubgraphsSaveMutationVariables = Exact<{
  input: SetTgChatSubgraphsInput;
}>;


export type AdminTgbotsShowchatsSubgraphsSaveMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'SetTgChatSubgraphsPayload', success: boolean } } };

export type AdminTgBotInviteChatsQueryVariables = Exact<{
  filter: AdminTgBotChatsFilterInput;
}>;


export type AdminTgBotInviteChatsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', tgBotChats: { __typename?: 'AdminTgBotChatsConnection', nodes: Array<{ __typename?: 'AdminTgBotChat', id: any, chatType: string, chatTitle: string, addedAt: any, removedAt?: any | null, memberCount: number, subgraphInvites: Array<{ __typename?: 'AdminTgBotChatSubgraphInvite', id: string, subgraphId: any }> }> } } };

export type AdminTgbotShowInviteChatsSubgraphsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminTgbotShowInviteChatsSubgraphsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename?: 'AdminSubgraph', id: any, name: string }> } } };

export type AdminTgbotShowInviteChatsSubgraphsSaveMutationVariables = Exact<{
  input: SetTgChatSubgraphInvitesInput;
}>;


export type AdminTgbotShowInviteChatsSubgraphsSaveMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'SetTgChatSubgraphInvitesPayload', success: boolean } } };

export type AdminTgbotShowPublishInstantTagsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminTgbotShowPublishInstantTagsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allTelegramPublishTags: { __typename?: 'AdminTelegramPublishTagsConnection', nodes: Array<{ __typename?: 'AdminTelegramPublishTag', id: any, label: string }> } } };

export type AdminTgbotShowPublishInstantTagsSaveMutationVariables = Exact<{
  input: SetTgChatPublishInstantTagsInput;
}>;


export type AdminTgbotShowPublishInstantTagsSaveMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'SetTgChatPublishInstantTagsPayload', success: boolean } } };

export type AdminTgBotPublishTagsQueryVariables = Exact<{
  filter: AdminTgBotChatsFilterInput;
}>;


export type AdminTgBotPublishTagsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', tgBotChats: { __typename?: 'AdminTgBotChatsConnection', nodes: Array<{ __typename?: 'AdminTgBotChat', id: any, chatType: string, chatTitle: string, addedAt: any, removedAt?: any | null, memberCount: number, publishTags: Array<{ __typename?: 'AdminTelegramPublishTag', id: any }>, publishInstantTags: Array<{ __typename?: 'AdminTelegramPublishTag', id: any }> }> } } };

export type AdminTgbotShowPublishTagsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminTgbotShowPublishTagsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allTelegramPublishTags: { __typename?: 'AdminTelegramPublishTagsConnection', nodes: Array<{ __typename?: 'AdminTelegramPublishTag', id: any, label: string }> } } };

export type AdminTgbotShowPublishTagsSaveMutationVariables = Exact<{
  input: SetTgChatPublishTagsInput;
}>;


export type AdminTgbotShowPublishTagsSaveMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'SetTgChatPublishTagsPayload', success: boolean } } };

export type AdminShowTgBotQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminShowTgBotQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', tgBot?: { __typename?: 'AdminTgBot', id: any, name: string, description: string, enabled: boolean, createdAt: any, createdBy: { __typename?: 'AdminUser', email?: string | null } } | null } };

export type AdminUpdateTgBotMutationMutationVariables = Exact<{
  input: UpdateTgBotInput;
}>;


export type AdminUpdateTgBotMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'UpdateTgBotPayload', tgBot: { __typename?: 'AdminTgBot', id: any, description: string } } } };

export type AdminListUserBansQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminListUserBansQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allUserUserBans: { __typename?: 'AdminUserBansConnection', nodes: Array<{ __typename?: 'UserBan', createdAt: any, reason: string, id: any, user: { __typename: 'AdminUser', email?: string | null }, bannedBy?: { __typename?: 'Admin', user: { __typename?: 'AdminUser', email?: string | null } } | null }> } } };

export type AdminBanUserMutationVariables = Exact<{
  input: BanUserInput;
}>;


export type AdminBanUserMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', banUser: { __typename: 'BanUserPayload', user: { __typename: 'AdminUser', id: any } } | { __typename: 'ErrorPayload', message: string } } };

export type AdminUnbanUserMutationVariables = Exact<{
  input: UnbanUserInput;
}>;


export type AdminUnbanUserMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'UnbanUserPayload', user: { __typename: 'AdminUser', id: any } } } };

export type AdminListUsersQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminListUsersQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allUsers: { __typename?: 'AdminUsersConnection', nodes: Array<{ __typename?: 'AdminUser', id: any, email?: string | null, createdAt: any, ban?: { __typename?: 'UserBan', reason: string } | null }> } } };

export type AdminCreateUserMutationVariables = Exact<{
  input: CreateUserInput;
}>;


export type AdminCreateUserMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', createUser: { __typename?: 'CreateUserPayload', user: { __typename?: 'AdminUser', id: any } } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminUserShowQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminUserShowQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', user?: { __typename?: 'AdminUser', id: any, email?: string | null, createdAt: any } | null } };

export type AdminUserSubgraphAccessQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminUserSubgraphAccessQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename?: 'AdminSubgraph', id: any, name: string }> }, userSubgraphAccess?: { __typename?: 'AdminUserSubgraphAccess', userId: any, subgraphId: any, expiresAt?: any | null } | null } };

export type AdminUpdateUserSubgraphAccessMutationVariables = Exact<{
  input: UpdateUserSubgraphAccessInput;
}>;


export type AdminUpdateUserSubgraphAccessMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'UpdateUserSubgraphAccessPayload', userSubgraphAccess: { __typename: 'UserSubgraphAccess', expiresAt?: any | null } } } };

export type AdminListUserSubgraphAccessesQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminListUserSubgraphAccessesQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', data: { __typename?: 'AdminUserSubgraphAccessesConnection', nodes: Array<{ __typename: 'AdminUserSubgraphAccess', id: any, createdAt: any, expiresAt?: any | null, subgraph: { __typename?: 'AdminSubgraph', name: string }, user: { __typename?: 'AdminUser', id: any, email?: string | null } }> } } };

export type AdminUserEditQueryQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminUserEditQueryQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', user?: { __typename?: 'AdminUser', id: any, email?: string | null, createdAt: any } | null } };

export type AdminUpdateUserMutationVariables = Exact<{
  input: UpdateUserInput;
}>;


export type AdminUpdateUserMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', updateUser: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'UpdateUserPayload', user: { __typename?: 'AdminUser', id: any, email?: string | null } } } };

export type AdminWaitListEmailRequestsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminWaitListEmailRequestsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allWaitListEmailRequests: { __typename?: 'AdminWaitListEmailRequestsConnection', nodes: Array<{ __typename?: 'AdminWaitListEmailRequest', email: string, createdAt: any, ip?: string | null, notePath: string }> } } };

export type AdminWaitListTgBotRequestsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminWaitListTgBotRequestsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allWaitListTgBotRequests: { __typename?: 'AdminWaitListTgBotRequestsConnection', nodes: Array<{ __typename?: 'AdminWaitListTgBotRequest', chatId: any, createdAt: any, notePathId: any, notePath: string, botName: string }> } } };

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


export type ReaderQueryQuery = { __typename?: 'Query', note?: { __typename?: 'PublicNote', title: string, html: string, pathId: any, toc: Array<{ __typename?: 'NoteTocItem', id: string, title: string, level: number }> } | null };

export type FavoriteNotesQueryVariables = Exact<{ [key: string]: never; }>;


export type FavoriteNotesQuery = { __typename?: 'Query', viewer: { __typename?: 'Viewer', user?: { __typename?: 'User', favoriteNotes: Array<{ __typename?: 'PublicNote', pathId: any }> } | null } };

export type ToggleFavoriteNoteMutationVariables = Exact<{
  input: ToggleFavoriteNoteInput;
}>;


export type ToggleFavoriteNoteMutation = { __typename?: 'Mutation', payload: { __typename?: 'ErrorPayload', message: string } | { __typename?: 'ToggleFavoriteNotePayload', favoriteNotes: Array<{ __typename?: 'PublicNote', pathId: any }> } };

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

export type SiteSearchQueryVariables = Exact<{
  input: SearchInput;
}>;


export type SiteSearchQuery = { __typename?: 'Query', search: { __typename?: 'SearchConnection', nodes: Array<{ __typename?: 'SearchResult', highlightedTitle?: string | null, highlightedContent: Array<string>, id: string }> } };

export type UserSubscriptionsQueryVariables = Exact<{ [key: string]: never; }>;


export type UserSubscriptionsQuery = { __typename?: 'Query', viewer: { __typename?: 'Viewer', user?: { __typename?: 'User', subgraphAccesses: Array<{ __typename?: 'UserSubgraphAccess', id: string, createdAt: any, expiresAt?: any | null, subgraph: { __typename?: 'Subgraph', name: string, homePath: string } }> } | null } };

export function $trip2g_graphql_request(query: '\n\t\tquery Admins {\n\t\t\tadmin {\n\t\t\t\tallAdmins {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tgrantedAt\n\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation DisableApiKey($input: DisableApiKeyInput!) {\n\t\t\tadmin {\n\t\t\t\tdata: disableApiKey(input: $input) {\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on DisableApiKeyPayload {\n\t\t\t\t\t\tapiKey {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: DisableApiKeyMutationVariables) => DisableApiKeyMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminListApiKeys {\n\t\t\tadmin {\n\t\t\t\tallApiKeys {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tdescription\n\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t\tdisabledAt\n\t\t\t\t\t\tdisabledBy {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminListApiKeysQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminCreateApiKey($input: CreateApiKeyInput!) {\n\t\t\tadmin {\n\t\t\t\tdata: createApiKey(input: $input) {\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on CreateApiKeyPayload {\n\t\t\t\t\t\tvalue\n\t\t\t\t\t\tapiKey {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCreateApiKeyMutationVariables) => AdminCreateApiKeyMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminApiKeyShowQuery($filter: ApiKeyLogsFilterInput!) {\n\t\t\tadmin {\n\t\t\t\tapiKeyLogs(filter: $filter) {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tactionName\n\t\t\t\t\t\tip\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminApiKeyShowQueryQueryVariables) => AdminApiKeyShowQueryQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminAuditLogs($filter: AdminAuditLogsFilterInput!) {\n\t\t\tadmin {\n\t\t\t\tauditLogs(filter: $filter) {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tlevel\n\t\t\t\t\t\tmessage\n\t\t\t\t\t\tparams\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminAuditLogsQueryVariables) => AdminAuditLogsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminDeleteBoostyCredentials($input: DeleteBoostyCredentialsInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: deleteBoostyCredentials(input: $input) {\n\t\t\t\t\t... on ErrorPayload{\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on DeleteBoostyCredentialsPayload {\n\t\t\t\t\t\tboostyCredentials {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminDeleteBoostyCredentialsMutationVariables) => AdminDeleteBoostyCredentialsMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation RefreshBoostyData($input: RefreshBoostyDataInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: refreshBoostyData(input: $input) {\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on RefreshBoostyDataPayload {\n\t\t\t\t\t\tsuccess\n\t\t\t\t\t\tcredentials {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: RefreshBoostyDataMutationVariables) => RefreshBoostyDataMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminRestoreBoostyCredentials($input: RestoreBoostyCredentialsInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: restoreBoostyCredentials(input: $input) {\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on RestoreBoostyCredentialsPayload {\n\t\t\t\t\t\tboostyCredentials {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminRestoreBoostyCredentialsMutationVariables) => AdminRestoreBoostyCredentialsMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminBoostyCredentials($filter: AdminBoostyCredentialsFilterInput) {\n\t\t\tadmin {\n\t\t\t\tallBoostyCredentials(filter: $filter) {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tstate\n\t\t\t\t\t\tdeviceId\n\t\t\t\t\t\tblogName\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminBoostyCredentialsQueryVariables) => AdminBoostyCredentialsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminCreateBoostyCreds($input: CreateBoostyCredentialsInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: createBoostyCredentials(input: $input) {\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on CreateBoostyCredentialsPayload {\n\t\t\t\t\t\tboostyCredentials {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCreateBoostyCredsMutationVariables) => AdminCreateBoostyCredsMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminBoostyCredentialsById($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\tboostyCredentials(id: $id) {\n\t\t\t\t\tcreatedAt\n\t\t\t\t\tdeviceId\n\t\t\t\t\tblogName\n\t\t\t\t\tstate\n\n\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\temail\n\t\t\t\t\t}\n\n\t\t\t\t\ttiers {\n\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tname\n\n\t\t\t\t\t\t\tsubgraphs {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t\n\t\t\t\t\tmembers {\n\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\tstatus\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminBoostyCredentialsByIdQueryVariables) => AdminBoostyCredentialsByIdQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminBoostycredentialsShowSubgraphs {\n\t\t\tadmin {\n\t\t\t\tallSubgraphs {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tname\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminBoostycredentialsShowSubgraphsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminBoostycredentialsShowSubgraphsSave($input: SetBoostyTierSubgraphsInput!) {\n\t\t\tadmin {\n\t\t\t\tdata: setBoostyTierSubgraphs(input: $input) {\n\t\t\t\t\t... on SetBoostyTierSubgraphsPayload {\n\t\t\t\t\t\tsuccess\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminBoostycredentialsShowSubgraphsSaveMutationVariables) => AdminBoostycredentialsShowSubgraphsSaveMutation

export function $trip2g_graphql_request(query: '\n\t\t\tquery AdminConfigVersions {\n\t\t\t\tadmin {\n\t\t\t\t\tallConfigVersions {\n\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\tshowDraftVersions\n\t\t\t\t\t\t\tdefaultLayout\n\t\t\t\t\t\t\ttimezone\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): () => AdminConfigVersionsQuery

export function $trip2g_graphql_request(query: '\n\t\t\tquery AdminCreateConfigLatestConfig {\n\t\t\t\tadmin {\n\t\t\t\t\tlatestConfig {\n\t\t\t\t\t\tshowDraftVersions\n\t\t\t\t\t\tdefaultLayout\n\t\t\t\t\t\ttimezone\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): () => AdminCreateConfigLatestConfigQuery

export function $trip2g_graphql_request(query: '\n\t\t\tmutation AdminCreateConfigVersion($input: CreateConfigVersionInput!) {\n\t\t\t\tadmin {\n\t\t\t\t\tdata: createConfigVersion(input: $input) {\n\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t\t... on CreateConfigVersionPayload {\n\t\t\t\t\t\t\tconfigVersion {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): (variables: AdminCreateConfigVersionMutationVariables) => AdminCreateConfigVersionMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminRunCronJob($input: RunCronJobInput!) {\n\t\t\tadmin {\n\t\t\t\trunCronJob(input: $input) {\n\t\t\t\t\t... on RunCronJobPayload {\n\t\t\t\t\t\texecution {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tjob {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminRunCronJobMutationVariables) => AdminRunCronJobMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminAllCronJobs {\n\t\t\tadmin {\n\t\t\t\tallCronJobs {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tname\n\t\t\t\t\t\tenabled\n\t\t\t\t\t\texpression\n\t\t\t\t\t\tlastExecAt\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminAllCronJobsQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminCronJobExecutions($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\tcronJob(id: $id) {\n\t\t\t\t\tid\n\t\t\t\t\texecutions {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tstartedAt\n\t\t\t\t\t\tfinishedAt\n\t\t\t\t\t\tstatus\n\t\t\t\t\t\terrorMessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCronJobExecutionsQueryVariables) => AdminCronJobExecutionsQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminCronJobShow($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\tcronJob(id: $id) {\n\t\t\t\t\tid\n\t\t\t\t\tname\n\t\t\t\t\tenabled\n\t\t\t\t\texpression\n\t\t\t\t\tlastExecAt\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCronJobShowQueryVariables) => AdminCronJobShowQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminCronJobUpdate($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\tcronJob(id: $id) {\n\t\t\t\t\tid\n\t\t\t\t\tname\n\t\t\t\t\tenabled\n\t\t\t\t\texpression\n\t\t\t\t\tlastExecAt\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCronJobUpdateQueryVariables) => AdminCronJobUpdateQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminUpdateCronJob($input: UpdateCronJobInput!) {\n\t\t\tadmin {\n\t\t\t\tupdateCronJob(input: $input) {\n\t\t\t\t\t... on UpdateCronJobPayload {\n\t\t\t\t\t\tcronJob {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\texpression\n\t\t\t\t\t\t\tenabled\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminUpdateCronJobMutationVariables) => AdminUpdateCronJobMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation DisableGitToken($input: DisableGitTokenInput!) {\n\t\t\tadmin {\n\t\t\t\tdata: disableGitToken(input: $input) {\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on DisableGitTokenPayload {\n\t\t\t\t\t\tgitToken {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: DisableGitTokenMutationVariables) => DisableGitTokenMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminGitTokens {\n\t\t\tadmin {\n\t\t\t\tallGitTokens {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tdescription\n\t\t\t\t\t\tcanPull\n\t\t\t\t\t\tcanPush\n\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t\tdisabledAt\n\t\t\t\t\t\tdisabledBy {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminGitTokensQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminCreateGitToken($input: CreateGitTokenInput!) {\n\t\t\tadmin {\n\t\t\t\tdata: createGitToken(input: $input) {\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on CreateGitTokenPayload {\n\t\t\t\t\t\tvalue\n\t\t\t\t\t\tgitToken {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCreateGitTokenMutationVariables) => AdminCreateGitTokenMutation

export function $trip2g_graphql_request(query: '\n\t\t\tmutation AdminDeleteHtmlInjection($input: DeleteHtmlInjectionInput!) {\n\t\t\t\tadmin {\n\t\t\t\t\tdata: deleteHtmlInjection(input: $input) {\n\t\t\t\t\t\t... on DeleteHtmlInjectionPayload {\n\t\t\t\t\t\t\tdeletedId\n\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t}\n\n\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): (variables: AdminDeleteHtmlInjectionMutationVariables) => AdminDeleteHtmlInjectionMutation

export function $trip2g_graphql_request(query: '\n\t\t\tquery AdminHtmlInjections {\n\t\t\t\tadmin {\n\t\t\t\t\tallHtmlInjections {\n\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\tactiveFrom\n\t\t\t\t\t\t\tactiveTo\n\t\t\t\t\t\t\tdescription\n\t\t\t\t\t\t\tposition\n\t\t\t\t\t\t\tplacement\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): () => AdminHtmlInjectionsQuery

export function $trip2g_graphql_request(query: '\n\t\t\tmutation AdminCreateHtmlInjectionMutation($input: CreateHtmlInjectionInput!) {\n\t\t\t\tadmin {\n\t\t\t\t\tdata: createHtmlInjection(input: $input) {\n\t\t\t\t\t\t... on CreateHtmlInjectionPayload {\n\t\t\t\t\t\t\thtmlInjection {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): (variables: AdminCreateHtmlInjectionMutationMutationVariables) => AdminCreateHtmlInjectionMutationMutation

export function $trip2g_graphql_request(query: '\n\t\t\tquery AdminShowHtmlInjection($id: Int64!) {\n\t\t\t\tadmin {\n\t\t\t\t\thtmlInjection(id: $id) {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tactiveFrom\n\t\t\t\t\t\tactiveTo\n\t\t\t\t\t\tdescription\n\t\t\t\t\t\tposition\n\t\t\t\t\t\tplacement\n\t\t\t\t\t\tcontent\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): (variables: AdminShowHtmlInjectionQueryVariables) => AdminShowHtmlInjectionQuery

export function $trip2g_graphql_request(query: '\n\t\t\tquery AdminUpdateDataHtmlInjection($id: Int64!) {\n\t\t\t\tadmin {\n\t\t\t\t\thtmlInjection(id: $id) {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tactiveFrom\n\t\t\t\t\t\tactiveTo\n\t\t\t\t\t\tdescription\n\t\t\t\t\t\tposition\n\t\t\t\t\t\tplacement\n\t\t\t\t\t\tcontent\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): (variables: AdminUpdateDataHtmlInjectionQueryVariables) => AdminUpdateDataHtmlInjectionQuery

export function $trip2g_graphql_request(query: '\n\t\t\tmutation AdminUpdateHtmlInjection($input: UpdateHtmlInjectionInput!) {\n\t\t\t\tadmin {\n\t\t\t\t\tdata: updateHtmlInjection(input: $input) {\n\t\t\t\t\t\t... on UpdateHtmlInjectionPayload {\n\t\t\t\t\t\t\thtmlInjection {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): (variables: AdminUpdateHtmlInjectionMutationVariables) => AdminUpdateHtmlInjectionMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminNoteAssets {\n\t\t\tadmin {\n\t\t\t\tallLatestNoteAssets {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tabsolutePath\n\t\t\t\t\t\tfileName\n\t\t\t\t\t\tsize\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminNoteAssetsQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminNoteAsset($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\tnoteAsset(id: $id) {\n\t\t\t\t\tid\n\t\t\t\t\tabsolutePath\n\t\t\t\t\tfileName\n\t\t\t\t\tsize\n\t\t\t\t\tcreatedAt\n\t\t\t\t\turl\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminNoteAssetQueryVariables) => AdminNoteAssetQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminListNoteViews {\n\t\t\tadmin {\n\t\t\t\tallLatestNoteViews {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tpath\n\t\t\t\t\t\ttitle\n\t\t\t\t\t\tfree\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminListNoteViewsQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminGraph {\n\t\t\tadmin {\n\t\t\t\tallSubgraphs {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tname\n\t\t\t\t\t\tcolor\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t\tallLatestNoteViews {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tsubgraphNames\n\t\t\t\t\t\ttitle\n\t\t\t\t\t\tpathId\n\t\t\t\t\t\tfree\n\t\t\t\t\t\tisHomePage\n\t\t\t\t\t\tgraphPosition{\n\t\t\t\t\t\t\tx,\n\t\t\t\t\t\t\ty,\n\t\t\t\t\t\t}\n\t\t\t\t\t\tinLinks {\n\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t\tpathId\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n'): () => AdminGraphQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation UpdateNoteGraphPositions($input: UpdateNoteGraphPositionsInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: updateNoteGraphPositions(input: $input) {\n\t\t\t\t\t... on UpdateNoteGraphPositionsPayload {\n\t\t\t\t\t\tsuccess\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: UpdateNoteGraphPositionsMutationVariables) => UpdateNoteGraphPositionsMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminSelectNoteView {\n\t\t\tadmin {\n\t\t\t\tallLatestNoteViews {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tversionId\n\t\t\t\t\t\tpath\n\t\t\t\t\t\ttitle\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminSelectNoteViewQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminNoteView($id: String!) {\n\t\t\tadmin {\n\t\t\t\tnoteView(id: $id) {\n\t\t\t\t\t__typename\n\t\t\t\t\tpath\n\t\t\t\t\ttitle\n\t\t\t\t\tpermalink\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminNoteViewQueryVariables) => AdminNoteViewQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminNoteWarnings($filter: AdminLatestNoteViewsFilter) {\n\t\t\tadmin {\n\t\t\t\tallLatestNoteViews(filter: $filter) {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tpath\n\t\t\t\t\t\twarnings {\n\t\t\t\t\t\t\tlevel\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminNoteWarningsQueryVariables) => AdminNoteWarningsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminResetNotFoundPath($input: ResetNotFoundPathInput!) {\n\t\t\tadmin {\n\t\t\t\tdata: resetNotFoundPath(input: $input) {\n\t\t\t\t\t... on ResetNotFoundPathPayload {\n\t\t\t\t\t\tnotFoundPath {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminResetNotFoundPathMutationVariables) => AdminResetNotFoundPathMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminNotFoundPaths {\n\t\t\tadmin {\n\t\t\t\tallNotFoundPaths {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tpath\n\t\t\t\t\t\ttotalHits\n\t\t\t\t\t\tlastHitAt\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminNotFoundPathsQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminShowNotFoundPath {\n\t\t\tadmin {\n\t\t\t\tallNotFoundPaths {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tpath\n\t\t\t\t\t\ttotalHits\n\t\t\t\t\t\tlastHitAt\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminShowNotFoundPathQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminDeleteNotFoundIgnoredPattern($input: DeleteNotFoundIgnoredPatternInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: deleteNotFoundIgnoredPattern(input: $input) {\n\t\t\t\t\t... on DeleteNotFoundIgnoredPatternPayload {\n\t\t\t\t\t\tdeletedId\n\t\t\t\t\t\t__typename\n\t\t\t\t\t}\n\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminDeleteNotFoundIgnoredPatternMutationVariables) => AdminDeleteNotFoundIgnoredPatternMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminNotFoundIgnoredPatterns {\n\t\t\tadmin {\n\t\t\t\tallNotFoundIgnoredPatterns {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tpattern\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminNotFoundIgnoredPatternsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminCreateNotFoundIgnoredPatternMutation($input: CreateNotFoundIgnoredPatternInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: createNotFoundIgnoredPattern(input: $input) {\n\t\t\t\t\t... on CreateNotFoundIgnoredPatternPayload {\n\t\t\t\t\t\tnotFoundIgnoredPattern {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCreateNotFoundIgnoredPatternMutationMutationVariables) => AdminCreateNotFoundIgnoredPatternMutationMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminShowNotFoundIgnoredPattern {\n\t\t\tadmin {\n\t\t\t\tallNotFoundIgnoredPatterns {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tpattern\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminShowNotFoundIgnoredPatternQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminDeleteNotFoundIgnoredPatternMutation($input: DeleteNotFoundIgnoredPatternInput!) {\n\t\t\tadmin {\n\t\t\t\tdata: deleteNotFoundIgnoredPattern(input: $input) {\n\t\t\t\t\t... on DeleteNotFoundIgnoredPatternPayload {\n\t\t\t\t\t\tdeletedId\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminDeleteNotFoundIgnoredPatternMutationMutationVariables) => AdminDeleteNotFoundIgnoredPatternMutationMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminUpdateNotFoundIgnoredPatternMutation($input: UpdateNotFoundIgnoredPatternInput!) {\n\t\t\tadmin {\n\t\t\t\tdata: updateNotFoundIgnoredPattern(input: $input) {\n\t\t\t\t\t... on UpdateNotFoundIgnoredPatternPayload {\n\t\t\t\t\t\tnotFoundIgnoredPattern {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminUpdateNotFoundIgnoredPatternMutationMutationVariables) => AdminUpdateNotFoundIgnoredPatternMutationMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminOffers {\n\t\t\tadmin {\n\t\t\t\tallOffers {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tpublicId\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tlifetime\n\t\t\t\t\t\tpriceUSD\n\t\t\t\t\t\tstartsAt\n\t\t\t\t\t\tendsAt\n\t\t\t\t\t\tsubgraphs {\n\t\t\t\t\t\t\tname\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminOffersQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminCreateOfferMutation($input: CreateOfferInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: createOffer(input: $input) {\n\t\t\t\t\t... on CreateOfferPayload {\n\t\t\t\t\t\toffer {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tpublicId\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCreateOfferMutationMutationVariables) => AdminCreateOfferMutationMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminShowOffer($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\toffer(id: $id) {\n\t\t\t\t\tid\n\t\t\t\t\tpublicId\n\t\t\t\t\tcreatedAt\n\t\t\t\t\tlifetime\n\t\t\t\t\tpriceUSD\n\t\t\t\t\tstartsAt\n\t\t\t\t\tendsAt\n\t\t\t\t\tsubgraphIds\n\t\t\t\t\tsubgraphs {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tname\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminShowOfferQueryVariables) => AdminShowOfferQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminUpdateOfferMutation($input: UpdateOfferInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: updateOffer(input: $input) {\n\t\t\t\t\t... on UpdateOfferPayload {\n\t\t\t\t\t\toffer {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tpublicId\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminUpdateOfferMutationMutationVariables) => AdminUpdateOfferMutationMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminDeletePatreonCredentials($input: DeletePatreonCredentialsInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: deletePatreonCredentials(input: $input) {\n\t\t\t\t\t... on ErrorPayload{\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on DeletePatreonCredentialsPayload {\n\t\t\t\t\t\tpatreonCredentials {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminDeletePatreonCredentialsMutationVariables) => AdminDeletePatreonCredentialsMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation RefreshPatreonData($input: RefreshPatreonDataInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: refreshPatreonData(input: $input) {\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on RefreshPatreonDataPayload {\n\t\t\t\t\t\tsuccess\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: RefreshPatreonDataMutationVariables) => RefreshPatreonDataMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminRestorePatreonCredentials($input: RestorePatreonCredentialsInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: restorePatreonCredentials(input: $input) {\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on RestorePatreonCredentialsPayload {\n\t\t\t\t\t\tpatreonCredentials {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminRestorePatreonCredentialsMutationVariables) => AdminRestorePatreonCredentialsMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminPatreonCredentials($filter: AdminPatreonCredentialsFilterInput) {\n\t\t\tadmin {\n\t\t\t\tallPatreonCredentials(filter: $filter) {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tstate\n\t\t\t\t\t\tcreatorAccessToken\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tsyncedAt\n\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminPatreonCredentialsQueryVariables) => AdminPatreonCredentialsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminCreatePatreonCreds($input: CreatePatreonCredentialsInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: createPatreonCredentials(input: $input) {\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on CreatePatreonCredentialsPayload {\n\t\t\t\t\t\tpatreonCredentials {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCreatePatreonCredsMutationVariables) => AdminCreatePatreonCredsMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminPatreonCredentialsById($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\tpatreonCredentials(id: $id) {\n\t\t\t\t\tcreatedAt\n\t\t\t\t\tcreatorAccessToken\n\t\t\t\t\tstate\n\n\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\temail\n\t\t\t\t\t}\n\n\t\t\t\t\ttiers {\n\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tmissedAt\n\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t\tamountCents\n\n\t\t\t\t\t\t\tsubgraphs {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t\n\t\t\t\t\tmembers {\n\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\tstatus\n\t\t\t\t\t\tcurrentTier {\n\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t}\n'): (variables: AdminPatreonCredentialsByIdQueryVariables) => AdminPatreonCredentialsByIdQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminPatreoncredentialsShowSubgraphs {\n\t\t\tadmin {\n\t\t\t\tallSubgraphs {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tname\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminPatreoncredentialsShowSubgraphsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminPatreoncredentialsShowSubgraphsSave($input: SetPatreonTierSubgraphsInput!) {\n\t\t\tadmin {\n\t\t\t\tdata: setPatreonTierSubgraphs(input: $input) {\n\t\t\t\t\t... on SetPatreonTierSubgraphsPayload {\n\t\t\t\t\t\tsuccess\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminPatreoncredentialsShowSubgraphsSaveMutationVariables) => AdminPatreoncredentialsShowSubgraphsSaveMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminPurchases {\n\t\t\tadmin {\n\t\t\t\tallPurchases {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tpaymentProvider\n\t\t\t\t\t\tstatus\n\t\t\t\t\t\tsuccessful\n\t\t\t\t\t\tofferId\n\t\t\t\t\t\temail\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminPurchasesQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminRedirects {\n\t\t\tadmin {\n\t\t\t\tallRedirects {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tpattern\n\t\t\t\t\t\tignoreCase\n\t\t\t\t\t\tisRegex\n\t\t\t\t\t\ttarget\n\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminRedirectsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminCreateRedirectMutation($input: CreateRedirectInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: createRedirect(input: $input) {\n\t\t\t\t\t... on CreateRedirectPayload {\n\t\t\t\t\t\tredirect {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCreateRedirectMutationMutationVariables) => AdminCreateRedirectMutationMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminShowRedirect($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\tredirect(id: $id) {\n\t\t\t\t\tid\n\t\t\t\t\tcreatedAt\n\t\t\t\t\tpattern\n\t\t\t\t\tignoreCase\n\t\t\t\t\tisRegex\n\t\t\t\t\ttarget\n\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\tid\n\t\t\t\t\t\temail\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminShowRedirectQueryVariables) => AdminShowRedirectQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminDeleteRedirectMutation($input: DeleteRedirectInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: deleteRedirect(input: $input) {\n\t\t\t\t\t... on DeleteRedirectPayload {\n\t\t\t\t\t\tid\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminDeleteRedirectMutationMutationVariables) => AdminDeleteRedirectMutationMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminUpdateRedirectMutation($input: UpdateRedirectInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: updateRedirect(input: $input) {\n\t\t\t\t\t... on UpdateRedirectPayload {\n\t\t\t\t\t\tredirect {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminUpdateRedirectMutationMutationVariables) => AdminUpdateRedirectMutationMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminMakeReleaseLive($input: MakeReleaseLiveInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: makeReleaseLive(input:$input) {\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on MakeReleaseLivePayload {\n\t\t\t\t\t\trelease {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminMakeReleaseLiveMutationVariables) => AdminMakeReleaseLiveMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminReleases {\n\t\t\tadmin {\n\t\t\t\tallReleases {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tcreatedBy{\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t\ttitle\n\t\t\t\t\t\tisLive\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminReleasesQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminCreateRelease($input: CreateReleaseInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: createRelease(input: $input) {\n\t\t\t\t\t... on CreateReleasePayload {\n\t\t\t\t\t\trelease {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCreateReleaseMutationVariables) => AdminCreateReleaseMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminListSubgraphs {\n\t\t\tadmin {\n\t\t\t\tallSubgraphs {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\t__typename\n\t\t\t\t\t\tid\n\t\t\t\t\t\tname\n\t\t\t\t\t\tcolor\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminListSubgraphsQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminSelectSubgraphList {\n\t\t\tadmin {\n\t\t\t\tallSubgraphs {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tname\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminSelectSubgraphListQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminSelectSubgraph {\n\t\t\tadmin {\n\t\t\t\tallSubgraphs {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tname\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminSelectSubgraphQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminShowSubgraph($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\tsubgraph(id: $id) {\n\t\t\t\t\tid\n\t\t\t\t\tname\n\t\t\t\t\tcolor\n\t\t\t\t\thidden\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminShowSubgraphQueryVariables) => AdminShowSubgraphQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation UpdateSubgraph($input: UpdateSubgraphInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: updateSubgraph(input: $input) {\n\t\t\t\t\t... on UpdateSubgraphPayload {\n\t\t\t\t\t\t__typename\n\t\t\t\t\t\tsubgraph {\n\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tcolor\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t__typename\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: UpdateSubgraphMutationVariables) => UpdateSubgraphMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminResetTelegramPublishNote($input: ResetTelegramPublishNoteInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: resetTelegramPublishNote(input: $input) {\n\t\t\t\t\t... on ResetTelegramPublishNotePayload {\n\t\t\t\t\t\tpublishNote {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminResetTelegramPublishNoteMutationVariables) => AdminResetTelegramPublishNoteMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminTelegramPublishNoteCount($filter: AdminTelegramPublishNotesFilter!) {\n\t\t\tadmin {\n\t\t\t\tallTelegramPublishNotes(filter: $filter) {\n\t\t\t\t\tcount\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminTelegramPublishNoteCountQueryVariables) => AdminTelegramPublishNoteCountQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminTelegramPublishNotes($filter: AdminTelegramPublishNotesFilter!) {\n\t\t\tadmin {\n\t\t\t\tallTelegramPublishNotes(filter: $filter) {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tpublishAt\n\t\t\t\t\t\tsecondsUntilPublish\n\t\t\t\t\t\tpublishedAt\n\t\t\t\t\t\tstatus\n\t\t\t\t\t\tnoteView {\n\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminTelegramPublishNotesQueryVariables) => AdminTelegramPublishNotesQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminTelegramPublishNote($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\ttelegramPublishNote(id: $id) {\n\t\t\t\t\tid\n\t\t\t\t\tcreatedAt\n\t\t\t\t\tpublishAt\n\t\t\t\t\tsecondsUntilPublish\n\t\t\t\t\tpublishedAt\n\t\t\t\t\tstatus\n\t\t\t\t\ttags {\n\t\t\t\t\t\tlabel\n\t\t\t\t\t}\n\t\t\t\t\tchats {\n\t\t\t\t\t\tchatTitle\n\t\t\t\t\t\tchatType\n\t\t\t\t\t}\n\t\t\t\t\tnoteView {\n\t\t\t\t\t\ttitle\n\t\t\t\t\t}\n\t\t\t\t\tpost {\n\t\t\t\t\t\tcontent\n\t\t\t\t\t\twarnings\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminTelegramPublishNoteQueryVariables) => AdminTelegramPublishNoteQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminTgBots {\n\t\t\tadmin {\n\t\t\t\tallTgBots {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tname\n\t\t\t\t\t\tdescription\n\t\t\t\t\t\tenabled\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminTgBotsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminCreateTgBotMutation($input: CreateTgBotInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: createTgBot(input: $input) {\n\t\t\t\t\t... on CreateTgBotPayload {\n\t\t\t\t\t\ttgBot {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tname\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCreateTgBotMutationMutationVariables) => AdminCreateTgBotMutationMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminTgBotChats($filter: AdminTgBotChatsFilterInput!) {\n\t\t\tadmin {\n\t\t\t\ttgBotChats(filter: $filter) {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tchatType\n\t\t\t\t\t\tchatTitle\n\t\t\t\t\t\taddedAt\n\t\t\t\t\t\tremovedAt\n\t\t\t\t\t\tmemberCount\n\t\t\t\t\t\tsubgraphAccesses {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tsubgraphId\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminTgBotChatsQueryVariables) => AdminTgBotChatsQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminTgbotShowChatsSubgraphs {\n\t\t\tadmin {\n\t\t\t\tallSubgraphs {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tname\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminTgbotShowChatsSubgraphsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminTgbotsShowchatsSubgraphsSave($input: SetTgChatSubgraphsInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: setTgChatSubgraphs(input: $input) {\n\t\t\t\t\t... on SetTgChatSubgraphsPayload {\n\t\t\t\t\t\tsuccess\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminTgbotsShowchatsSubgraphsSaveMutationVariables) => AdminTgbotsShowchatsSubgraphsSaveMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminTgBotInviteChats($filter: AdminTgBotChatsFilterInput!) {\n\t\t\tadmin {\n\t\t\t\ttgBotChats(filter: $filter) {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tchatType\n\t\t\t\t\t\tchatTitle\n\t\t\t\t\t\taddedAt\n\t\t\t\t\t\tremovedAt\n\t\t\t\t\t\tmemberCount\n\t\t\t\t\t\tsubgraphInvites {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tsubgraphId\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminTgBotInviteChatsQueryVariables) => AdminTgBotInviteChatsQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminTgbotShowInviteChatsSubgraphs {\n\t\t\tadmin {\n\t\t\t\tallSubgraphs {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tname\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminTgbotShowInviteChatsSubgraphsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminTgbotShowInviteChatsSubgraphsSave($input: SetTgChatSubgraphInvitesInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: setTgChatSubgraphInvites(input: $input) {\n\t\t\t\t\t... on SetTgChatSubgraphInvitesPayload {\n\t\t\t\t\t\tsuccess\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminTgbotShowInviteChatsSubgraphsSaveMutationVariables) => AdminTgbotShowInviteChatsSubgraphsSaveMutation

export function $trip2g_graphql_request(query: '\n\t\t\tquery AdminTgbotShowPublishInstantTags {\n\t\t\t\tadmin {\n\t\t\t\t\tallTelegramPublishTags {\n\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tlabel\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): () => AdminTgbotShowPublishInstantTagsQuery

export function $trip2g_graphql_request(query: '\n\t\t\tmutation AdminTgbotShowPublishInstantTagsSave($input: SetTgChatPublishInstantTagsInput!) {\n\t\t\t\tadmin {\n\t\t\t\t\tdata: setTgChatPublishInstantTags(input: $input) {\n\t\t\t\t\t\t... on SetTgChatPublishInstantTagsPayload {\n\t\t\t\t\t\t\tsuccess\n\t\t\t\t\t\t}\n\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): (variables: AdminTgbotShowPublishInstantTagsSaveMutationVariables) => AdminTgbotShowPublishInstantTagsSaveMutation

export function $trip2g_graphql_request(query: '\n\t\t\tquery AdminTgBotPublishTags($filter: AdminTgBotChatsFilterInput!) {\n\t\t\t\tadmin {\n\t\t\t\t\ttgBotChats(filter: $filter) {\n\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tchatType\n\t\t\t\t\t\t\tchatTitle\n\t\t\t\t\t\t\taddedAt\n\t\t\t\t\t\t\tremovedAt\n\t\t\t\t\t\t\tmemberCount\n\t\t\t\t\t\t\tpublishTags {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\tpublishInstantTags {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): (variables: AdminTgBotPublishTagsQueryVariables) => AdminTgBotPublishTagsQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminTgbotShowPublishTags {\n\t\t\tadmin {\n\t\t\t\tallTelegramPublishTags {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tlabel\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminTgbotShowPublishTagsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminTgbotShowPublishTagsSave($input: SetTgChatPublishTagsInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: setTgChatPublishTags(input: $input) {\n\t\t\t\t\t... on SetTgChatPublishTagsPayload {\n\t\t\t\t\t\tsuccess\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminTgbotShowPublishTagsSaveMutationVariables) => AdminTgbotShowPublishTagsSaveMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminShowTgBot($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\ttgBot(id: $id) {\n\t\t\t\t\tid\n\t\t\t\t\tname\n\t\t\t\t\tdescription\n\t\t\t\t\tenabled\n\t\t\t\t\tcreatedAt\n\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\temail\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminShowTgBotQueryVariables) => AdminShowTgBotQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminUpdateTgBotMutation($input: UpdateTgBotInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: updateTgBot(input: $input) {\n\t\t\t\t\t... on UpdateTgBotPayload {\n\t\t\t\t\t\ttgBot {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tdescription\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminUpdateTgBotMutationMutationVariables) => AdminUpdateTgBotMutationMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminListUserBans {\n\t\t\tadmin {\n\t\t\t\tallUserUserBans {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid: userId\n\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t\tbannedBy {\n\t\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\treason\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminListUserBansQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminBanUser($input: BanUserInput!) {\n\t\t\tadmin {\n\t\t\t\tbanUser(input: $input) {\n\t\t\t\t\t... on BanUserPayload {\n\t\t\t\t\t\t__typename\n\t\t\t\t\t\tuser { id, __typename }\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t__typename\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminBanUserMutationVariables) => AdminBanUserMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminUnbanUser($input: UnbanUserInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: unbanUser(input: $input) {\n\t\t\t\t\t... on UnbanUserPayload {\n\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminUnbanUserMutationVariables) => AdminUnbanUserMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminListUsers {\n\t\t\tadmin {\n\t\t\t\tallUsers {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\temail\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tban { reason }\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminListUsersQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminCreateUser($input: CreateUserInput!) {\n\t\t\tadmin {\n\t\t\t\tcreateUser(input: $input) {\n\t\t\t\t\t... on CreateUserPayload {\n\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCreateUserMutationVariables) => AdminCreateUserMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminUserShow($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\tuser(id: $id) {\n\t\t\t\t\tid\n\t\t\t\t\temail\n\t\t\t\t\tcreatedAt\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminUserShowQueryVariables) => AdminUserShowQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminUserSubgraphAccess($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\tallSubgraphs {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tname\n\t\t\t\t\t}\n\t\t\t\t}\n\n\t\t\t\tuserSubgraphAccess(id: $id) {\n\t\t\t\t\tuserId\n\t\t\t\t\tsubgraphId\n\t\t\t\t\texpiresAt\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminUserSubgraphAccessQueryVariables) => AdminUserSubgraphAccessQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminUpdateUserSubgraphAccess($input: UpdateUserSubgraphAccessInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: updateUserSubgraphAccess(input: $input) {\n\t\t\t\t\t... on UpdateUserSubgraphAccessPayload {\n\t\t\t\t\t\tuserSubgraphAccess {\n\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\texpiresAt\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminUpdateUserSubgraphAccessMutationVariables) => AdminUpdateUserSubgraphAccessMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminListUserSubgraphAccesses {\n\t\t\tadmin {\n\t\t\t\tdata: allUserSubgraphAccesses {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\t__typename\n\t\t\t\t\t\tid\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\texpiresAt\n\t\t\t\t\t\tsubgraph {\n\t\t\t\t\t\t\tname\n\t\t\t\t\t\t}\n\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminListUserSubgraphAccessesQuery

export function $trip2g_graphql_request(query: '\n\t\t\tquery AdminUserEditQuery($id: Int64!) {\n\t\t\t\tadmin {\n\t\t\t\t\tuser(id: $id) {\n\t\t\t\t\t\tid\n\t\t\t\t\t\temail\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): (variables: AdminUserEditQueryQueryVariables) => AdminUserEditQueryQuery

export function $trip2g_graphql_request(query: '\n\t\t\tmutation AdminUpdateUser($input: UpdateUserInput!) {\n\t\t\t\tadmin {\n\t\t\t\t\tupdateUser(input: $input) {\n\t\t\t\t\t\t... on UpdateUserPayload {\n\t\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t}\n\t\t}\n\t\t'): (variables: AdminUpdateUserMutationVariables) => AdminUpdateUserMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminWaitListEmailRequests {\n\t\t\tadmin {\n\t\t\t\tallWaitListEmailRequests {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\temail\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tip\n\t\t\t\t\t\tnotePath\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminWaitListEmailRequestsQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminWaitListTgBotRequests {\n\t\t\tadmin {\n\t\t\t\tallWaitListTgBotRequests {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tchatId\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tnotePathId\n\t\t\t\t\t\tnotePath\n\t\t\t\t\t\tbotName\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminWaitListTgBotRequestsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation SignOut {\n\t\t\tdata: signOut {\n\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t__typename\n\t\t\t\t\tmessage\n\t\t\t\t}\n\t\t\t\t... on SignOutPayload {\n\t\t\t\t\t__typename\n\t\t\t\t\tviewer {\n\t\t\t\t\t\tid\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => SignOutMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation RequestEmailSignInCode($input: RequestEmailSignInCodeInput!) {\n\t\t\tdata: requestEmailSignInCode(input: $input) {\n\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t__typename\n\t\t\t\t\tmessage\n\t\t\t\t}\n\t\t\t\t... on RequestEmailSignInCodePayload {\n\t\t\t\t\t__typename\n\t\t\t\t\tsuccess\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: RequestEmailSignInCodeMutationVariables) => RequestEmailSignInCodeMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation SignInByEmail($input: SignInByEmailInput!) {\n\t\t\tdata: signInByEmail(input: $input) {\n\t\t\t\t... on SignInPayload {\n\t\t\t\t\t__typename\n\t\t\t\t\ttoken\n\t\t\t\t}\n\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t__typename\n\t\t\t\t\tmessage\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: SignInByEmailMutationVariables) => SignInByEmailMutation

export function $trip2g_graphql_request(query: '\n\t\tquery Viewer {\n\t\t\tviewer {\n\t\t\t\tid\n\t\t\t\trole\n\t\t\t\tuser {\n\t\t\t\t\temail\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => ViewerQuery

export function $trip2g_graphql_request(query: '\n\t\tquery ReaderQuery($input: NoteInput!) {\n\t\t\tnote(input: $input) {\n\t\t\t\ttitle\n\t\t\t\thtml\n\t\t\t\tpathId\n\t\t\t\ttoc {\n\t\t\t\t\tid\n\t\t\t\t\ttitle\n\t\t\t\t\tlevel\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: ReaderQueryQueryVariables) => ReaderQueryQuery

export function $trip2g_graphql_request(query: '\n\t\tquery FavoriteNotes {\n\t\t\tviewer {\n\t\t\t\tuser {\n\t\t\t\t\tfavoriteNotes {\n\t\t\t\t\t\tpathId\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => FavoriteNotesQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation ToggleFavoriteNote($input: ToggleFavoriteNoteInput!) {\n\t\t\tpayload: toggleFavoriteNote(input: $input) {\n\t\t\t\t... on ToggleFavoriteNotePayload {\n\t\t\t\t\tfavoriteNotes {\n\t\t\t\t\t\tpathId\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t\t... on ErrorPayload {\n\t\t\t\t\tmessage\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: ToggleFavoriteNoteMutationVariables) => ToggleFavoriteNoteMutation

export function $trip2g_graphql_request(query: '\n\t\tquery PaywallActivePurchaseQuery {\n\t\t\tviewer {\n\t\t\t\tactivePurchases {\n\t\t\t\t\tid\n\t\t\t\t\tstatus\n\t\t\t\t\tsuccessful\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => PaywallActivePurchaseQueryQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation CreateEmailWaitListRequestMutation ($input: CreateEmailWaitListRequestInput!) {\n\t\t\tcreateEmailWaitListRequest(input: $input) {\n\t\t\t\t... on ErrorPayload {\n\t\t\t\t\tmessage\n\t\t\t\t}\n\t\t\t\t... on CreateEmailWaitListRequestPayload {\n\t\t\t\t\tsuccess\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: CreateEmailWaitListRequestMutationMutationVariables) => CreateEmailWaitListRequestMutationMutation

export function $trip2g_graphql_request(query: '\n\t\tquery PaywallQuery($filter: ViewerOffersFilter!) {\n\t\t\tviewer {\n\t\t\t\toffers(filter: $filter) {\n\t\t\t\t\t... on ActiveOffers {\n\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tpriceUSD\n\t\t\t\t\t\t\tsubgraphs {\n\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on SubgraphWaitList {\n\t\t\t\t\t\ttgBotUrl\n\t\t\t\t\t\temailAllowed\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: PaywallQueryQueryVariables) => PaywallQueryQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation CreatePaymentLink($input: CreatePaymentLinkInput!) {\n\t\t\tdata: createPaymentLink(input: $input) {\n\t\t\t\t... on CreatePaymentLinkPayload {\n\t\t\t\t\tredirectUrl\n\t\t\t\t}\n\t\t\t\t... on ErrorPayload {\n\t\t\t\t\tmessage\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: CreatePaymentLinkMutationVariables) => CreatePaymentLinkMutation

export function $trip2g_graphql_request(query: '\n\t\tquery SiteSearch($input: SearchInput!) {\n\t\t\tsearch(input: $input) {\n\t\t\t\tnodes {\n\t\t\t\t\thighlightedTitle\n\t\t\t\t\thighlightedContent\n\t\t\t\t\tid: url\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: SiteSearchQueryVariables) => SiteSearchQuery

export function $trip2g_graphql_request(query: '\n\t\tquery UserSubscriptions {\n\t\t\tviewer {\n\t\t\t\tuser {\n\t\t\t\t\tsubgraphAccesses {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\texpiresAt\n\t\t\t\t\t\tsubgraph {\n\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\thomePath\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => UserSubscriptionsQuery

export function $trip2g_graphql_request(query: any) { return $trip2g_graphql_raw_request(query); }

export function $trip2g_graphql_subscription(query: any, variables?: any) { return $trip2g_graphql_raw_subscription(query, variables); }



export const $trip2g_graphql_persist_queries = {"Admins":"2209bde3b712b460384f1dd9af9b34a08e48b0919881f04bea07eb6de7e55157","DisableApiKey":"47a9cf3dd02fe35a6ad13c0a7ea8c4e5c30e7bd66ae513ce8b8505784a017fb9","AdminListApiKeys":"13383a7d8aec7f1ed51c53e633afda42678bae49018418dcd44b0d8d4fc96934","AdminCreateApiKey":"235670c9dfa7e81b6742a93f752efb3f28dc6fa264a81719a5aeffd7af609d9f","AdminApiKeyShowQuery":"94e1e99e9f9f05d96115e8edabf88e7bb9cd995106966d9dc54859f822031ebb","AdminAuditLogs":"7732bb7b9eefd49612edb50b516fb64a8e9e4bf7783f5ea1cdc35413862521e9","AdminDeleteBoostyCredentials":"579d8eb6aeb88488760a569482a8a4a1e83f0adb971e61179fb80bfc961d84d7","RefreshBoostyData":"863fb2eacc48044c0e2dc82b616163adb360d8c0920a8f15ecae854443121890","AdminRestoreBoostyCredentials":"012a8d3ef47f26ab6894d2b5684f3ea963d3c1f15da73ad5d7006b477b541d70","AdminBoostyCredentials":"4f86130ad2ca88640244c90cdbf75368f5c83a36760f73bf85a1c1aaadbd6549","AdminCreateBoostyCreds":"f99081c86753d0819dbfc35fb5f00df070f326c4cfd1c4b4f517e7b81527a4a7","AdminBoostyCredentialsById":"25d361c2ec04bd37e335aa6cb6127651538ff746cb2aae68d57c331c49cc627e","AdminBoostycredentialsShowSubgraphs":"ba441dfe401c8a2567e6bee71a7e1d82a5131aaf0accb914c0616171dfb6b14a","AdminBoostycredentialsShowSubgraphsSave":"b54fdd63e7ab61271dec2ce1de9ae25097c75ecc5c93471cf23e70a8df4b9760","AdminConfigVersions":"5e50260c7c49ba2bc77db663d9dea7ec6dd93519afdaee059adad89089a9e79f","AdminCreateConfigLatestConfig":"c95e80d95fcb2d78fe738b77d9054b028b81cf40bb2ff556a5761e11a4d2bdb0","AdminCreateConfigVersion":"22f12dd703a49ed7e9ead9951d76579752ae8ab9fc3166247b331d7ade2d8b42","AdminRunCronJob":"caafefaf93ce047b1bb760d517cf37ce359f23bbe3cb810d342a18ef68fd0111","AdminAllCronJobs":"3d6356462a6f2ef6022d0b35c0367a76790db754a9a4aced549f00978a7078c2","AdminCronJobExecutions":"8cf2f57f809f108bea8232b50e42a140a402b83193b8e3f67a8fd8e5a48ae594","AdminCronJobShow":"bd777a26fe99c1c995773da5e53e89d1cf3792a541cf4d8f2b5a692bf73250a7","AdminCronJobUpdate":"ff19b1f3c3aa76479671a28fe6e7acdfe6b80bdfe0fd66b2d922c6f49c6f8f92","AdminUpdateCronJob":"9a2cb2dbd0714de71e20e7d82b1bb649ed770175d3288493a4f2f0fd53ca716e","DisableGitToken":"e88ba1aefe8fdc597a8ebc126ecbdd058d6f08f2e0c57540678062434dd3c583","AdminGitTokens":"eb696170eb5fc39c34851e905fde6f8fb4fd67c01379e002c5679ee04254ba88","AdminCreateGitToken":"01be653e006d3dc95721b92f14c0defb46e51eb422cf672410f8d1051881d376","AdminDeleteHtmlInjection":"02825d38c5fcba8ac342ec99a13b2c5004956cb16a642f56f7ea6bb208062cb7","AdminHtmlInjections":"73fd86c77f13e9b01e8fc121522f4f010292fc46c1783555f2352334f5e910a2","AdminCreateHtmlInjectionMutation":"fbe272a0604d63d59a0d891b1d46d6cc1d5577d695e4ee7d63e1b22a0d0fdb31","AdminShowHtmlInjection":"a93a473968b1c40f55fdbd6d1f668768d257e3f5811dde5ce729eff967b9eab2","AdminUpdateDataHtmlInjection":"a02c7a970320c0c4b31373419a982712708baa199ab92ec45fb17c35802dc667","AdminUpdateHtmlInjection":"928b2959c3f29b30779996e8fd8836fb6587e9ffd1688b2814ae73a29b1cf949","AdminNoteAssets":"087e5fc30451d6207eda1bdb1e03f213145f02cb1cb7d95a14b6efe56908d4a4","AdminNoteAsset":"44eedd49083501e533b8ad22c6a10120796a73a2de0a364e48b074a74d994fd2","AdminListNoteViews":"109e1088644d1a890d18dca41a2a24575d0576f423bbb8465e3341f6b8426627","AdminGraph":"11e1135806a03eeefb9837d0aa9d66f9ca01cc07c98de4c6aefc59b413b15ea3","UpdateNoteGraphPositions":"e6a7ba49f82328d3565612928b4228ded8b987dcbd08c052b770d21454414e5a","AdminSelectNoteView":"8d41af85838e81d47ac73b1e162eb57aade00d338f6b245d4e2ce5032ca14dfe","AdminNoteView":"1b4435167f9c9481d61439d49505a5323b42140a8abe8bca7a2468198e07a05e","AdminNoteWarnings":"7afa7d4a98c2575d5ce092aea4be74a168d2204051106a361cfda3a5f26ae574","AdminResetNotFoundPath":"1c6838c3c1f9303fc5669151ca681ceb9516cc487182556e0cce3bf3cf98e790","AdminNotFoundPaths":"282f9ee7b0b3685acfef7a58e953281618c86b599d1feec39c8130e775c61030","AdminShowNotFoundPath":"324127af97d96a755c5e8a8d51eb34a1a8d6e97a0b76edd98e5342b4d72744dd","AdminDeleteNotFoundIgnoredPattern":"25df6d8595f77b4c6015b88c7ef5d932e1638ec35fb83783f531ea6dec5ddec1","AdminNotFoundIgnoredPatterns":"991caef559d9083af0f8d9e52c75c3e7542c61ba6beb40b4889002c83229fbb4","AdminCreateNotFoundIgnoredPatternMutation":"df2ea2e10aed4e13ac818ebe331280cc376d5a389524cb421ccdb4e48214dc0c","AdminShowNotFoundIgnoredPattern":"d0e687ab326ae5ca1c8a8f1fc6d5309d301cd8e9397e4dbb4cd9f241c9f0f2ec","AdminDeleteNotFoundIgnoredPatternMutation":"5a6b1e7708a9de675a077c6ae759cd97963d18cc5449029f964fbee17c325537","AdminUpdateNotFoundIgnoredPatternMutation":"fa6c52b727a7cf3cad7911e9abbb50e427168e1c885d3330b4188d3d541b4220","AdminOffers":"3cad705de7d0a6f946bc5500877f873a9ab25bc7342d03e237c9b359afaa3629","AdminCreateOfferMutation":"a19ec8cef5bb93b4999e09ba1a294b46659c77a301a8ff32c09f0e4b1a50b672","AdminShowOffer":"cbf19b83b7bd0731904cb4ce6788ee70bad5f491db944db43f4c34163b0b8fc5","AdminUpdateOfferMutation":"476d44296aa62d6390d2ccf2421a85507ab49bb9fdbb31ad53e39b9f08b79287","AdminDeletePatreonCredentials":"5568e61c12f062dcf47ddbdfcde9e0ec10869d1faee5cff15d2baf83948cfed8","RefreshPatreonData":"05cf524e046d207f546c95e1e8cc259408a1966aa2b24858fbf071d45f4ca5fd","AdminRestorePatreonCredentials":"44bf4c73e3c8e7300ea618a81b2815cf954aac591023bb59a9b3156d40480cab","AdminPatreonCredentials":"d24d17b73dfe39800dad5dcd5be1d6213a8800f34b4938f23063901e01955764","AdminCreatePatreonCreds":"067743c634074b854e76a8815062c722e826d7d70533b92c8ab522d4f1feb4f9","AdminPatreonCredentialsById":"8d54a2c27a8c6c67f6848a71df667206d6648da78f26dcd43c8382aa95bb5e58","AdminPatreoncredentialsShowSubgraphs":"b6e4e79f24364a6b5915a1e368112fbf80d9dbc3bbb1f79b2e3fc7e2daac415e","AdminPatreoncredentialsShowSubgraphsSave":"5226d6f658f13a5405e17d757dedd9b406b6f9006d758a3007363b635699a33e","AdminPurchases":"ef072cc649880d869c6583e74a8fcb2babc3885784f30c6de0c155061c4d6b5a","AdminRedirects":"8f2e44663bb975f3d859e68461531283b75d2012733073124c7ca2b239f90195","AdminCreateRedirectMutation":"489461b00ff6568a2fc6e2628a0539f8f9a50201dc0adfcfb7efc99222bc420f","AdminShowRedirect":"e1aff6da682993975d38be08ed4d23ac822cae5d5cccc440adb81709096541d5","AdminDeleteRedirectMutation":"0e21d691458fa1d848330e11411526b9a7dbb02f18fab96de4cfe54e287f71ae","AdminUpdateRedirectMutation":"3795e3067ff2bcb4e2b99513e3eb121cb39f8b411e94b490e998b6301c275fcb","AdminMakeReleaseLive":"6b629eac886998de002fb9752c58ceac7148c5adb3577114fb31dcc54548702c","AdminReleases":"268a822109d34a08b6842c0cdf9fc9c4affa9ebcb631c13a0063776c875d0453","AdminCreateRelease":"50cf5039d5e642773d65b4af39fb433cefa62f78d108bb07d59da7b9033dbf95","AdminListSubgraphs":"736583bdaabf701bb4c263550d07343474039f696a8abdfa73c85d6e9d498504","AdminSelectSubgraphList":"27f8a22ce0111c5300586855183f08c383a28dc0f3a03a1e7a90f9546a561e9e","AdminSelectSubgraph":"bfb18e30a9fd494f178f47958f9577d5fd48a69bd5ab3e8b2e266c0fd5cbdc12","AdminShowSubgraph":"de6493bf83e6aa65609e71c0ac345de0a4d659e801ffba971578d340dd0c584c","UpdateSubgraph":"94f6835acbfe0449fa3e704756c35a39fbfeb4bc07c5e1942362ef163fb48851","AdminResetTelegramPublishNote":"b0bdda16b8d92571888c25dbede36902950c553b50787178bd738dd4319e7fba","AdminTelegramPublishNoteCount":"4d4684b8925974cce47f8be4543ca50830caa2c6477294ff1f4bd692f00a9950","AdminTelegramPublishNotes":"61a58e2e8656fe590117f90cf06d486969237f4af8ed7005f3e5bfa8e5c9473c","AdminTelegramPublishNote":"190ef3faf6dd9c7cb513fdc44b7a0acc850ade9f4867de58b4b0dd709fb289f1","AdminTgBots":"b28805b2174ede41d3d78afb3a9d65d9b895f397b592b34354588460f8e2bb42","AdminCreateTgBotMutation":"6489d744996f133923cbcd0e3774addfec985ba0aa9bdc6a5d736edab459c09a","AdminTgBotChats":"4109addb65bfd0519dadac163d49a9d75a4531864626b89f2be8e2c0459059d4","AdminTgbotShowChatsSubgraphs":"0dd960356eebc403c01bef443b820c9baae38fd3b4cbe69d439b810e11a3fb9d","AdminTgbotsShowchatsSubgraphsSave":"52877c37562e8725d030b7fcc7947299a58836a80d5860c7bddbb7a939449694","AdminTgBotInviteChats":"f7861f43b7df246ff72c892382938aa132909f13f3cc9bedfeb3290f0fe6717c","AdminTgbotShowInviteChatsSubgraphs":"9bbde11bb05b659a81a36247823506385160462c47c749a1d180792ff0042268","AdminTgbotShowInviteChatsSubgraphsSave":"bc844817799efad8403004511bbd0f588d76b2b33d5cf7bbcbc492fb50a76750","AdminTgbotShowPublishInstantTags":"f29a63a133073428082fa58492824a77c6b4023cca4c31e9c211a506fa811fbf","AdminTgbotShowPublishInstantTagsSave":"696a3a3a74379cc3309adf01ef58019826ccb6edce8a23584971ac8ed6800a6a","AdminTgBotPublishTags":"712ca8c5c56b3254bd1b262926c3d5be5a3cfaa59e80c1fe3e4e165c9acf5c1b","AdminTgbotShowPublishTags":"3d187a7e159d83880373e5b7f3823e25115a00dd9b0e418b930afd1401d4d251","AdminTgbotShowPublishTagsSave":"0f0d0001c09a69dc304be4d8064a422535e89d20eb802b674e251e6db04837da","AdminShowTgBot":"3156d2c98a70c8f463c36466301a7794a322ae31bede87c7b412afa9161d568f","AdminUpdateTgBotMutation":"a96fd1c45e3563cf6ff263a96ac99704019520bc3187c9fa9d1d5805b5dc455c","AdminListUserBans":"deca5c1020d24a6addb618b1cefaeb34ce259be30e4d8c8d3fb23d602db026d8","AdminBanUser":"8d89aad3f55509073d5320271722766c40c63600dd7bcce37322932c446498ca","AdminUnbanUser":"64ef0b51f7c725b147fdea1c659890f1ec9561c16196ae6e0eb121e4b0fc0795","AdminListUsers":"d5efc49d4cfe6f4c701bfce895c90ec5656faa0546a594f097d5298169bea2a7","AdminCreateUser":"192c484f4c51b4974ef7dceaf3d1cb917784236c4b6e98de7395b3136d514a79","AdminUserShow":"42ab8ea3fb6a7f6d1d4636853bce1d24278bc711bbe544968c6efb5208e34b88","AdminUserSubgraphAccess":"e1b29051bd0ca65a74a6450f38cc08a1a592645b752fbcc2ff285be5f8b6c195","AdminUpdateUserSubgraphAccess":"605c72682734aa100f48039a686c9bd1129bd9642a8d8199f874712fb8819dbc","AdminListUserSubgraphAccesses":"a111150d2d6e387ffbdc6831372858f9ab50327916b322e813ab5c1e42a23e83","AdminUserEditQuery":"918bd97114f7a93d86d2ae44b82386d463b40e825634c15c462908231cbae9a1","AdminUpdateUser":"21470cbaa253c01782eb5eb09d6ce17cd616db55b6f2754deefcb4cc1ce3c153","AdminWaitListEmailRequests":"e7ed63b82dac84395cc6d7994b4df630437a6ede6f41eb75cf19ea4f84f60fd0","AdminWaitListTgBotRequests":"f7b5d92c20e47a2ff61dee1d787a96ac3cf2d025a572fafb6076aee54c726a8a","SignOut":"dd9e88509a14eb1e23e55505d86b2ffe1eb7f2d423f30e11ef91ca20c1a41350","RequestEmailSignInCode":"e801cb599a3b0b00ceffb4d5158ff8150cc963444f3cbcd8817523598353a5e8","SignInByEmail":"6f77d0af492ac3750da375bcf0ed5a427c60033b7391603b96ba350650320920","Viewer":"e435bf22f939c4d9e198599b7131db2ce543693c80ed1572f2fbe4cb4e7a92cc","ReaderQuery":"977402b222d490ec562f9c053d7e44762cd0b4c7ae9c63c2946f1a5327206269","FavoriteNotes":"41db3398fb04b0058667ed2b6e4de673bdded9ab20f4eb9203db91501405c291","ToggleFavoriteNote":"def2207b3411f8555b05569c1231e4b70caf0ec2a08c5ae105856ad276b1bc4a","PaywallActivePurchaseQuery":"83d5209519d5c9e6e5ba07ecdeb5a8c0f5384658314ef05e82adced366dcedeb","CreateEmailWaitListRequestMutation":"1611de1ffcdc8a3c5205fb064418400e79b44701244bcc776f6d000a2cb84045","PaywallQuery":"36248fc923e1f5150904428e83197a2eafb53411f14acc4eef859b91e60726fd","CreatePaymentLink":"58f1824142bdc06565676489786cf8a6fc4e2ba450712e9bcf3206477b56b99b","SiteSearch":"4dbdf4a51d0e89de7c167e0c923831347617308f5cb96e6ef43b4e1876e78f69","UserSubscriptions":"54befef1facac5f0642e9c90827e2d57bcf32ed4582c1e1243e890b6b14f307b"}

export const $trip2g_graphql_audit_log_level_enum = AuditLogLevelEnum;

export const $trip2g_graphql_boosty_credentials_state_enum = BoostyCredentialsStateEnum;

export const $trip2g_graphql_cron_job_execution_status = CronJobExecutionStatus;

export const $trip2g_graphql_note_warning_level_enum = NoteWarningLevelEnum;

export const $trip2g_graphql_patreon_credentials_state_enum = PatreonCredentialsStateEnum;

export const $trip2g_graphql_payment_type = PaymentType;

export const $trip2g_graphql_role = Role;

// Generated variable type declarations

export type $trip2g_graphql_DisableApiKeyVariables = DisableApiKeyMutationVariables

export type $trip2g_graphql_AdminCreateApiKeyVariables = AdminCreateApiKeyMutationVariables

export type $trip2g_graphql_AdminApiKeyShowQueryVariables = AdminApiKeyShowQueryQueryVariables

export type $trip2g_graphql_AdminAuditLogsVariables = AdminAuditLogsQueryVariables

export type $trip2g_graphql_AdminDeleteBoostyCredentialsVariables = AdminDeleteBoostyCredentialsMutationVariables

export type $trip2g_graphql_RefreshBoostyDataVariables = RefreshBoostyDataMutationVariables

export type $trip2g_graphql_AdminRestoreBoostyCredentialsVariables = AdminRestoreBoostyCredentialsMutationVariables

export type $trip2g_graphql_AdminBoostyCredentialsVariables = AdminBoostyCredentialsQueryVariables

export type $trip2g_graphql_AdminCreateBoostyCredsVariables = AdminCreateBoostyCredsMutationVariables

export type $trip2g_graphql_AdminBoostyCredentialsByIdVariables = AdminBoostyCredentialsByIdQueryVariables

export type $trip2g_graphql_AdminBoostycredentialsShowSubgraphsSaveVariables = AdminBoostycredentialsShowSubgraphsSaveMutationVariables

export type $trip2g_graphql_AdminCreateConfigVersionVariables = AdminCreateConfigVersionMutationVariables

export type $trip2g_graphql_AdminRunCronJobVariables = AdminRunCronJobMutationVariables

export type $trip2g_graphql_AdminCronJobExecutionsVariables = AdminCronJobExecutionsQueryVariables

export type $trip2g_graphql_AdminCronJobShowVariables = AdminCronJobShowQueryVariables

export type $trip2g_graphql_AdminCronJobUpdateVariables = AdminCronJobUpdateQueryVariables

export type $trip2g_graphql_AdminUpdateCronJobVariables = AdminUpdateCronJobMutationVariables

export type $trip2g_graphql_DisableGitTokenVariables = DisableGitTokenMutationVariables

export type $trip2g_graphql_AdminCreateGitTokenVariables = AdminCreateGitTokenMutationVariables

export type $trip2g_graphql_AdminDeleteHtmlInjectionVariables = AdminDeleteHtmlInjectionMutationVariables

export type $trip2g_graphql_AdminCreateHtmlInjectionMutationVariables = AdminCreateHtmlInjectionMutationMutationVariables

export type $trip2g_graphql_AdminShowHtmlInjectionVariables = AdminShowHtmlInjectionQueryVariables

export type $trip2g_graphql_AdminUpdateDataHtmlInjectionVariables = AdminUpdateDataHtmlInjectionQueryVariables

export type $trip2g_graphql_AdminUpdateHtmlInjectionVariables = AdminUpdateHtmlInjectionMutationVariables

export type $trip2g_graphql_AdminNoteAssetVariables = AdminNoteAssetQueryVariables

export type $trip2g_graphql_UpdateNoteGraphPositionsVariables = UpdateNoteGraphPositionsMutationVariables

export type $trip2g_graphql_AdminNoteViewVariables = AdminNoteViewQueryVariables

export type $trip2g_graphql_AdminNoteWarningsVariables = AdminNoteWarningsQueryVariables

export type $trip2g_graphql_AdminResetNotFoundPathVariables = AdminResetNotFoundPathMutationVariables

export type $trip2g_graphql_AdminDeleteNotFoundIgnoredPatternVariables = AdminDeleteNotFoundIgnoredPatternMutationVariables

export type $trip2g_graphql_AdminCreateNotFoundIgnoredPatternMutationVariables = AdminCreateNotFoundIgnoredPatternMutationMutationVariables

export type $trip2g_graphql_AdminDeleteNotFoundIgnoredPatternMutationVariables = AdminDeleteNotFoundIgnoredPatternMutationMutationVariables

export type $trip2g_graphql_AdminUpdateNotFoundIgnoredPatternMutationVariables = AdminUpdateNotFoundIgnoredPatternMutationMutationVariables

export type $trip2g_graphql_AdminCreateOfferMutationVariables = AdminCreateOfferMutationMutationVariables

export type $trip2g_graphql_AdminShowOfferVariables = AdminShowOfferQueryVariables

export type $trip2g_graphql_AdminUpdateOfferMutationVariables = AdminUpdateOfferMutationMutationVariables

export type $trip2g_graphql_AdminDeletePatreonCredentialsVariables = AdminDeletePatreonCredentialsMutationVariables

export type $trip2g_graphql_RefreshPatreonDataVariables = RefreshPatreonDataMutationVariables

export type $trip2g_graphql_AdminRestorePatreonCredentialsVariables = AdminRestorePatreonCredentialsMutationVariables

export type $trip2g_graphql_AdminPatreonCredentialsVariables = AdminPatreonCredentialsQueryVariables

export type $trip2g_graphql_AdminCreatePatreonCredsVariables = AdminCreatePatreonCredsMutationVariables

export type $trip2g_graphql_AdminPatreonCredentialsByIdVariables = AdminPatreonCredentialsByIdQueryVariables

export type $trip2g_graphql_AdminPatreoncredentialsShowSubgraphsSaveVariables = AdminPatreoncredentialsShowSubgraphsSaveMutationVariables

export type $trip2g_graphql_AdminCreateRedirectMutationVariables = AdminCreateRedirectMutationMutationVariables

export type $trip2g_graphql_AdminShowRedirectVariables = AdminShowRedirectQueryVariables

export type $trip2g_graphql_AdminDeleteRedirectMutationVariables = AdminDeleteRedirectMutationMutationVariables

export type $trip2g_graphql_AdminUpdateRedirectMutationVariables = AdminUpdateRedirectMutationMutationVariables

export type $trip2g_graphql_AdminMakeReleaseLiveVariables = AdminMakeReleaseLiveMutationVariables

export type $trip2g_graphql_AdminCreateReleaseVariables = AdminCreateReleaseMutationVariables

export type $trip2g_graphql_AdminShowSubgraphVariables = AdminShowSubgraphQueryVariables

export type $trip2g_graphql_UpdateSubgraphVariables = UpdateSubgraphMutationVariables

export type $trip2g_graphql_AdminResetTelegramPublishNoteVariables = AdminResetTelegramPublishNoteMutationVariables

export type $trip2g_graphql_AdminTelegramPublishNoteCountVariables = AdminTelegramPublishNoteCountQueryVariables

export type $trip2g_graphql_AdminTelegramPublishNotesVariables = AdminTelegramPublishNotesQueryVariables

export type $trip2g_graphql_AdminTelegramPublishNoteVariables = AdminTelegramPublishNoteQueryVariables

export type $trip2g_graphql_AdminCreateTgBotMutationVariables = AdminCreateTgBotMutationMutationVariables

export type $trip2g_graphql_AdminTgBotChatsVariables = AdminTgBotChatsQueryVariables

export type $trip2g_graphql_AdminTgbotsShowchatsSubgraphsSaveVariables = AdminTgbotsShowchatsSubgraphsSaveMutationVariables

export type $trip2g_graphql_AdminTgBotInviteChatsVariables = AdminTgBotInviteChatsQueryVariables

export type $trip2g_graphql_AdminTgbotShowInviteChatsSubgraphsSaveVariables = AdminTgbotShowInviteChatsSubgraphsSaveMutationVariables

export type $trip2g_graphql_AdminTgbotShowPublishInstantTagsSaveVariables = AdminTgbotShowPublishInstantTagsSaveMutationVariables

export type $trip2g_graphql_AdminTgBotPublishTagsVariables = AdminTgBotPublishTagsQueryVariables

export type $trip2g_graphql_AdminTgbotShowPublishTagsSaveVariables = AdminTgbotShowPublishTagsSaveMutationVariables

export type $trip2g_graphql_AdminShowTgBotVariables = AdminShowTgBotQueryVariables

export type $trip2g_graphql_AdminUpdateTgBotMutationVariables = AdminUpdateTgBotMutationMutationVariables

export type $trip2g_graphql_AdminBanUserVariables = AdminBanUserMutationVariables

export type $trip2g_graphql_AdminUnbanUserVariables = AdminUnbanUserMutationVariables

export type $trip2g_graphql_AdminCreateUserVariables = AdminCreateUserMutationVariables

export type $trip2g_graphql_AdminUserShowVariables = AdminUserShowQueryVariables

export type $trip2g_graphql_AdminUserSubgraphAccessVariables = AdminUserSubgraphAccessQueryVariables

export type $trip2g_graphql_AdminUpdateUserSubgraphAccessVariables = AdminUpdateUserSubgraphAccessMutationVariables

export type $trip2g_graphql_AdminUserEditQueryVariables = AdminUserEditQueryQueryVariables

export type $trip2g_graphql_AdminUpdateUserVariables = AdminUpdateUserMutationVariables

export type $trip2g_graphql_RequestEmailSignInCodeVariables = RequestEmailSignInCodeMutationVariables

export type $trip2g_graphql_SignInByEmailVariables = SignInByEmailMutationVariables

export type $trip2g_graphql_ReaderQueryVariables = ReaderQueryQueryVariables

export type $trip2g_graphql_ToggleFavoriteNoteVariables = ToggleFavoriteNoteMutationVariables

export type $trip2g_graphql_CreateEmailWaitListRequestMutationVariables = CreateEmailWaitListRequestMutationMutationVariables

export type $trip2g_graphql_PaywallQueryVariables = PaywallQueryQueryVariables

export type $trip2g_graphql_CreatePaymentLinkVariables = CreatePaymentLinkMutationVariables

export type $trip2g_graphql_SiteSearchVariables = SiteSearchQueryVariables

}