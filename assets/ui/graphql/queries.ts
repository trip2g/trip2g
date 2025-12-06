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

export type AdminBackgroundJob = {
  __typename?: 'AdminBackgroundJob';
  id: Scalars['String']['output'];
  name: Scalars['String']['output'];
  params: Scalars['String']['output'];
  priority: Scalars['Int64']['output'];
  retryCount: Scalars['Int64']['output'];
};

export type AdminBackgroundQueue = {
  __typename?: 'AdminBackgroundQueue';
  id: Scalars['String']['output'];
  jobs: Array<AdminBackgroundJob>;
  pendingCount: Scalars['Int64']['output'];
  retryCount: Scalars['Int64']['output'];
  stopped: Scalars['Boolean']['output'];
};

export type AdminBackgroundQueuesConnection = {
  __typename?: 'AdminBackgroundQueuesConnection';
  nodes: Array<AdminBackgroundQueue>;
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

export type AdminCancelTelegramAccountAuthInput = {
  phone: Scalars['String']['input'];
};

export type AdminCancelTelegramAccountAuthOrErrorPayload = AdminCancelTelegramAccountAuthPayload | ErrorPayload;

export type AdminCancelTelegramAccountAuthPayload = {
  __typename?: 'AdminCancelTelegramAccountAuthPayload';
  success: Scalars['Boolean']['output'];
};

export type AdminCompleteTelegramAccountAuthInput = {
  code: Scalars['String']['input'];
  password?: InputMaybe<Scalars['String']['input']>;
  phone: Scalars['String']['input'];
};

export type AdminCompleteTelegramAccountAuthOrErrorPayload = AdminCompleteTelegramAccountAuthPayload | ErrorPayload;

export type AdminCompleteTelegramAccountAuthPayload = {
  __typename?: 'AdminCompleteTelegramAccountAuthPayload';
  account: AdminTelegramAccount;
};

export type AdminConfigVersion = {
  __typename?: 'AdminConfigVersion';
  createdAt: Scalars['Time']['output'];
  createdBy: AdminUser;
  defaultLayout: Scalars['String']['output'];
  id: Scalars['Int64']['output'];
  robotsTxt: Scalars['String']['output'];
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

export type AdminImportTelegramAccountChannelInput = {
  accountId: Scalars['Int64']['input'];
  basePath: Scalars['String']['input'];
  channelId: Scalars['Int64']['input'];
  skipExists?: InputMaybe<Scalars['Boolean']['input']>;
  withMedia?: InputMaybe<Scalars['Boolean']['input']>;
};

export type AdminImportTelegramAccountChannelOrErrorPayload = AdminImportTelegramAccountChannelPayload | ErrorPayload;

export type AdminImportTelegramAccountChannelPayload = {
  __typename?: 'AdminImportTelegramAccountChannelPayload';
  success: Scalars['Boolean']['output'];
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
  cancelTelegramAccountAuth: AdminCancelTelegramAccountAuthOrErrorPayload;
  clearBackgroundQueue: ClearBackgroundQueueOrErrorPayload;
  completeTelegramAccountAuth: AdminCompleteTelegramAccountAuthOrErrorPayload;
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
  importTelegramAccountChannel: AdminImportTelegramAccountChannelOrErrorPayload;
  makeReleaseLive: MakeReleaseLiveOrErrorPayload;
  refreshBoostyData: RefreshBoostyDataOrErrorPayload;
  refreshPatreonData: RefreshPatreonDataOrErrorPayload;
  removeExpiredTgChatMembers: RemoveExpiredTgChatMembersOrErrorPayload;
  resetNotFoundPath: ResetNotFoundPathOrErrorPayload;
  resetTelegramPublishNote: ResetTelegramPublishNoteOrErrorPayload;
  restoreBoostyCredentials: RestoreBoostyCredentialsOrErrorPayload;
  restorePatreonCredentials: RestorePatreonCredentialsOrErrorPayload;
  runCronJob: RunCronJobOrErrorPayload;
  sendTelegramPublishNoteNow: SendTelegramPublishNoteNowOrErrorPayload;
  setBoostyTierSubgraphs: SetBoostyTierSubgraphsOrErrorPayload;
  setPatreonTierSubgraphs: SetPatreonTierSubgraphsOrErrorPayload;
  setTelegramAccountChatPublishInstantTags: AdminSetTelegramAccountChatPublishInstantTagsOrErrorPayload;
  setTelegramAccountChatPublishTags: AdminSetTelegramAccountChatPublishTagsOrErrorPayload;
  setTgChatPublishInstantTags: SetTgChatPublishInstantTagsOrErrorPayload;
  setTgChatPublishTags: SetTgChatPublishTagsOrErrorPayload;
  setTgChatSubgraphInvites: SetTgChatSubgraphInvitesOrErrorPayload;
  setTgChatSubgraphs: SetTgChatSubgraphsOrErrorPayload;
  signOutTelegramAccount: AdminSignOutTelegramAccountOrErrorPayload;
  startBackgroundQueue: StartBackgroundQueueOrErrorPayload;
  startTelegramAccountAuth: AdminStartTelegramAccountAuthOrErrorPayload;
  stopBackgroundQueue: StopBackgroundQueueOrErrorPayload;
  unbanUser: UnbanUserOrErrorPayload;
  updateBoostyCredentials: UpdateBoostyCredentialsOrErrorPayload;
  updateCronJob: UpdateCronJobOrErrorPayload;
  updateHtmlInjection: UpdateHtmlInjectionOrErrorPayload;
  updateNotFoundIgnoredPattern: UpdateNotFoundIgnoredPatternOrErrorPayload;
  updateNoteGraphPositions: UpdateNoteGraphPositionsOrErrorPayload;
  updateOffer: UpdateOfferOrErrorPayload;
  updateRedirect: UpdateRedirectOrErrorPayload;
  updateSubgraph: UpdateSubgraphOrErrorPayload;
  updateTelegramAccount: AdminUpdateTelegramAccountOrErrorPayload;
  updateTgBot: UpdateTgBotOrErrorPayload;
  updateUser: UpdateUserOrErrorPayload;
  updateUserSubgraphAccess: UpdateUserSubgraphAccessOrErrorPayload;
};


export type AdminMutationBanUserArgs = {
  input: BanUserInput;
};


export type AdminMutationCancelTelegramAccountAuthArgs = {
  input: AdminCancelTelegramAccountAuthInput;
};


export type AdminMutationClearBackgroundQueueArgs = {
  input: ClearBackgroundQueueInput;
};


export type AdminMutationCompleteTelegramAccountAuthArgs = {
  input: AdminCompleteTelegramAccountAuthInput;
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


export type AdminMutationImportTelegramAccountChannelArgs = {
  input: AdminImportTelegramAccountChannelInput;
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


export type AdminMutationSendTelegramPublishNoteNowArgs = {
  input: SendTelegramPublishNoteNowInput;
};


export type AdminMutationSetBoostyTierSubgraphsArgs = {
  input: SetBoostyTierSubgraphsInput;
};


export type AdminMutationSetPatreonTierSubgraphsArgs = {
  input: SetPatreonTierSubgraphsInput;
};


export type AdminMutationSetTelegramAccountChatPublishInstantTagsArgs = {
  input: AdminSetTelegramAccountChatPublishInstantTagsInput;
};


export type AdminMutationSetTelegramAccountChatPublishTagsArgs = {
  input: AdminSetTelegramAccountChatPublishTagsInput;
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


export type AdminMutationSignOutTelegramAccountArgs = {
  input: AdminSignOutTelegramAccountInput;
};


export type AdminMutationStartBackgroundQueueArgs = {
  input: StartBackgroundQueueInput;
};


export type AdminMutationStartTelegramAccountAuthArgs = {
  input: AdminStartTelegramAccountAuthInput;
};


export type AdminMutationStopBackgroundQueueArgs = {
  input: StopBackgroundQueueInput;
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


export type AdminMutationUpdateTelegramAccountArgs = {
  input: AdminUpdateTelegramAccountInput;
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
  allBackgroundQueues: AdminBackgroundQueuesConnection;
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
  allTelegramAccounts: AdminTelegramAccountsConnection;
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
  backgroundQueue?: Maybe<AdminBackgroundQueue>;
  boostyCredentials?: Maybe<AdminBoostyCredentials>;
  buildGitCommit: Scalars['String']['output'];
  cronJob?: Maybe<AdminCronJob>;
  healthChecks: Array<HealchCheck>;
  htmlInjection?: Maybe<AdminHtmlInjection>;
  latestConfig: AdminConfigVersion;
  noteAsset?: Maybe<AdminNoteAsset>;
  noteView?: Maybe<NoteView>;
  offer?: Maybe<AdminOffer>;
  patreonCredentials?: Maybe<AdminPatreonCredentials>;
  purchase?: Maybe<AdminPurchase>;
  recentlyModifiedNoteViews: Array<NoteView>;
  redirect?: Maybe<AdminRedirect>;
  subgraph?: Maybe<AdminSubgraph>;
  telegramAccount?: Maybe<AdminTelegramAccount>;
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


export type AdminQueryBackgroundQueueArgs = {
  id: Scalars['String']['input'];
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


export type AdminQueryTelegramAccountArgs = {
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

export type AdminSetTelegramAccountChatPublishInstantTagsInput = {
  accountId: Scalars['Int64']['input'];
  tagIds: Array<Scalars['Int64']['input']>;
  telegramChatId: Scalars['String']['input'];
};

export type AdminSetTelegramAccountChatPublishInstantTagsOrErrorPayload = AdminSetTelegramAccountChatPublishInstantTagsPayload | ErrorPayload;

export type AdminSetTelegramAccountChatPublishInstantTagsPayload = {
  __typename?: 'AdminSetTelegramAccountChatPublishInstantTagsPayload';
  success: Scalars['Boolean']['output'];
};

export type AdminSetTelegramAccountChatPublishTagsInput = {
  accountId: Scalars['Int64']['input'];
  tagIds: Array<Scalars['Int64']['input']>;
  telegramChatId: Scalars['String']['input'];
};

export type AdminSetTelegramAccountChatPublishTagsOrErrorPayload = AdminSetTelegramAccountChatPublishTagsPayload | ErrorPayload;

export type AdminSetTelegramAccountChatPublishTagsPayload = {
  __typename?: 'AdminSetTelegramAccountChatPublishTagsPayload';
  success: Scalars['Boolean']['output'];
};

export type AdminSignOutTelegramAccountInput = {
  id: Scalars['Int64']['input'];
};

export type AdminSignOutTelegramAccountOrErrorPayload = AdminSignOutTelegramAccountPayload | ErrorPayload;

export type AdminSignOutTelegramAccountPayload = {
  __typename?: 'AdminSignOutTelegramAccountPayload';
  success: Scalars['Boolean']['output'];
};

export type AdminStartTelegramAccountAuthInput = {
  apiHash: Scalars['String']['input'];
  apiId: Scalars['Int']['input'];
  phone: Scalars['String']['input'];
};

export type AdminStartTelegramAccountAuthOrErrorPayload = AdminStartTelegramAccountAuthPayload | ErrorPayload;

export type AdminStartTelegramAccountAuthPayload = {
  __typename?: 'AdminStartTelegramAccountAuthPayload';
  authState: AdminTelegramAccountAuthState;
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

export type AdminTelegramAccount = {
  __typename?: 'AdminTelegramAccount';
  createdAt: Scalars['Time']['output'];
  createdBy?: Maybe<User>;
  dialogs: Array<AdminTelegramAccountDialog>;
  displayName: Scalars['String']['output'];
  enabled: Scalars['Boolean']['output'];
  id: Scalars['Int64']['output'];
  isPremium: Scalars['Boolean']['output'];
  phone: Scalars['String']['output'];
};

export type AdminTelegramAccountAuthState = {
  __typename?: 'AdminTelegramAccountAuthState';
  passwordHint?: Maybe<Scalars['String']['output']>;
  phone: Scalars['String']['output'];
  state: AdminTelegramAccountAuthStateEnum;
};

export enum AdminTelegramAccountAuthStateEnum {
  Authorized = 'AUTHORIZED',
  Error = 'ERROR',
  WaitingForCode = 'WAITING_FOR_CODE',
  WaitingForPassword = 'WAITING_FOR_PASSWORD'
}

export type AdminTelegramAccountDialog = {
  __typename?: 'AdminTelegramAccountDialog';
  id: Scalars['Int64']['output'];
  publishInstantTags: Array<AdminTelegramPublishTag>;
  publishTags: Array<AdminTelegramPublishTag>;
  title: Scalars['String']['output'];
  type: AdminTelegramAccountDialogType;
  username: Scalars['String']['output'];
};

export enum AdminTelegramAccountDialogType {
  Channel = 'channel',
  Chat = 'chat',
  User = 'user'
}

export type AdminTelegramAccountsConnection = {
  __typename?: 'AdminTelegramAccountsConnection';
  nodes: Array<AdminTelegramAccount>;
};

export type AdminTelegramPublishNote = {
  __typename?: 'AdminTelegramPublishNote';
  chats: Array<AdminTgBotChat>;
  createdAt: Scalars['Time']['output'];
  errorCount: Scalars['Int64']['output'];
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

export type AdminUpdateTelegramAccountInput = {
  displayName?: InputMaybe<Scalars['String']['input']>;
  enabled?: InputMaybe<Scalars['Boolean']['input']>;
  id: Scalars['Int64']['input'];
};

export type AdminUpdateTelegramAccountOrErrorPayload = AdminUpdateTelegramAccountPayload | ErrorPayload;

export type AdminUpdateTelegramAccountPayload = {
  __typename?: 'AdminUpdateTelegramAccountPayload';
  account: AdminTelegramAccount;
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

export type ClearBackgroundQueueInput = {
  id: Scalars['String']['input'];
};

export type ClearBackgroundQueueOrErrorPayload = ClearBackgroundQueuePayload | ErrorPayload;

export type ClearBackgroundQueuePayload = {
  __typename?: 'ClearBackgroundQueuePayload';
  deletedCount: Scalars['Int64']['output'];
  queue: AdminBackgroundQueue;
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
  robotsTxt: Scalars['String']['input'];
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

export type HealchCheck = {
  __typename?: 'HealchCheck';
  description: Scalars['String']['output'];
  id: Scalars['String']['output'];
  status: HealthCheckStatus;
};

export enum HealthCheckStatus {
  Critical = 'CRITICAL',
  Ok = 'OK',
  Warning = 'WARNING'
}

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

export type NoteAssetReplaceT = {
  __typename?: 'NoteAssetReplaceT';
  hash: Scalars['String']['output'];
  id: Scalars['String']['output'];
  url: Scalars['String']['output'];
};

export type NoteInput = {
  path?: InputMaybe<Scalars['String']['input']>;
  pathId?: InputMaybe<Scalars['Int64']['input']>;
  referer: Scalars['String']['input'];
};

export type NotePath = {
  __typename?: 'NotePath';
  latestContentHash: Scalars['String']['output'];
  latestNoteView?: Maybe<NoteView>;
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
  assetReplaces: Array<NoteAssetReplaceT>;
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

export type SendTelegramPublishNoteNowInput = {
  id: Scalars['Int64']['input'];
};

export type SendTelegramPublishNoteNowOrErrorPayload = ErrorPayload | SendTelegramPublishNoteNowPayload;

export type SendTelegramPublishNoteNowPayload = {
  __typename?: 'SendTelegramPublishNoteNowPayload';
  publishNote: AdminTelegramPublishNote;
};

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

export type StartBackgroundQueueInput = {
  id: Scalars['String']['input'];
};

export type StartBackgroundQueueOrErrorPayload = ErrorPayload | StartBackgroundQueuePayload;

export type StartBackgroundQueuePayload = {
  __typename?: 'StartBackgroundQueuePayload';
  queues: Array<AdminBackgroundQueue>;
};

export type StopBackgroundQueueInput = {
  id: Scalars['String']['input'];
};

export type StopBackgroundQueueOrErrorPayload = ErrorPayload | StopBackgroundQueuePayload;

export type StopBackgroundQueuePayload = {
  __typename?: 'StopBackgroundQueuePayload';
  queues: Array<AdminBackgroundQueue>;
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


export type DisableApiKeyMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename: 'DisableApiKeyPayload', apiKey: { __typename?: 'AdminApiKey', id: any } } | { __typename: 'ErrorPayload', message: string } } };

export type AdminListApiKeysQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminListApiKeysQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allApiKeys: { __typename?: 'AdminApiKeysConnection', nodes: Array<{ __typename?: 'AdminApiKey', id: any, createdAt: any, description: string, disabledAt?: any | null, createdBy: { __typename?: 'AdminUser', id: any, email?: string | null }, disabledBy?: { __typename?: 'AdminUser', id: any, email?: string | null } | null }> } } };

export type AdminCreateApiKeyMutationVariables = Exact<{
  input: CreateApiKeyInput;
}>;


export type AdminCreateApiKeyMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename: 'CreateApiKeyPayload', value: string, apiKey: { __typename?: 'AdminApiKey', id: any } } | { __typename: 'ErrorPayload', message: string } } };

export type AdminApiKeyShowQueryQueryVariables = Exact<{
  filter: ApiKeyLogsFilterInput;
}>;


export type AdminApiKeyShowQueryQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', apiKeyLogs: { __typename?: 'AdminApiKeyLogsConnection', nodes: Array<{ __typename?: 'AdminApiKeyLog', createdAt: any, actionName: string, ip: string }> } } };

export type AdminAuditLogsQueryVariables = Exact<{
  filter: AdminAuditLogsFilterInput;
}>;


export type AdminAuditLogsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', auditLogs: { __typename?: 'AdminAuditLogsConnection', nodes: Array<{ __typename?: 'AdminAuditLog', id: any, createdAt: any, level: AuditLogLevelEnum, message: string, params: string }> } } };

export type AdminClearBackgroundQueueMutationVariables = Exact<{
  input: ClearBackgroundQueueInput;
}>;


export type AdminClearBackgroundQueueMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'ClearBackgroundQueuePayload', deletedCount: any, queue: { __typename?: 'AdminBackgroundQueue', id: string, stopped: boolean } } | { __typename: 'ErrorPayload', message: string } } };

export type AdminStartBackgroundQueueMutationVariables = Exact<{
  input: StartBackgroundQueueInput;
}>;


export type AdminStartBackgroundQueueMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'ErrorPayload', message: string } | { __typename: 'StartBackgroundQueuePayload', queues: Array<{ __typename?: 'AdminBackgroundQueue', id: string, stopped: boolean }> } } };

export type AdminStopBackgroundQueueMutationVariables = Exact<{
  input: StopBackgroundQueueInput;
}>;


export type AdminStopBackgroundQueueMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'ErrorPayload', message: string } | { __typename: 'StopBackgroundQueuePayload', queues: Array<{ __typename?: 'AdminBackgroundQueue', id: string, stopped: boolean }> } } };

export type AdminBackgroundQueuesQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminBackgroundQueuesQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allBackgroundQueues: { __typename?: 'AdminBackgroundQueuesConnection', nodes: Array<{ __typename?: 'AdminBackgroundQueue', id: string, pendingCount: any, retryCount: any, stopped: boolean }> } } };

export type AdminBackgroundQueueQueryVariables = Exact<{
  id: Scalars['String']['input'];
}>;


export type AdminBackgroundQueueQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', backgroundQueue?: { __typename?: 'AdminBackgroundQueue', id: string, pendingCount: any, retryCount: any, stopped: boolean, jobs: Array<{ __typename?: 'AdminBackgroundJob', id: string, name: string, params: string, retryCount: any }> } | null } };

export type AdminDeleteBoostyCredentialsMutationVariables = Exact<{
  input: DeleteBoostyCredentialsInput;
}>;


export type AdminDeleteBoostyCredentialsMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'DeleteBoostyCredentialsPayload', boostyCredentials: { __typename?: 'AdminBoostyCredentials', id: any } } | { __typename: 'ErrorPayload', message: string } } };

export type RefreshBoostyDataMutationVariables = Exact<{
  input: RefreshBoostyDataInput;
}>;


export type RefreshBoostyDataMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'ErrorPayload', message: string } | { __typename: 'RefreshBoostyDataPayload', success: boolean, credentials: { __typename?: 'AdminBoostyCredentials', id: any } } } };

export type AdminRestoreBoostyCredentialsMutationVariables = Exact<{
  input: RestoreBoostyCredentialsInput;
}>;


export type AdminRestoreBoostyCredentialsMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'ErrorPayload', message: string } | { __typename: 'RestoreBoostyCredentialsPayload', boostyCredentials: { __typename?: 'AdminBoostyCredentials', id: any } } } };

export type AdminBoostyCredentialsQueryVariables = Exact<{
  filter?: InputMaybe<AdminBoostyCredentialsFilterInput>;
}>;


export type AdminBoostyCredentialsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allBoostyCredentials: { __typename?: 'AdminBoostyCredentialsConnection', nodes: Array<{ __typename?: 'AdminBoostyCredentials', id: any, state: BoostyCredentialsStateEnum, deviceId: string, blogName: string, createdAt: any, createdBy: { __typename?: 'AdminUser', id: any, email?: string | null } }> } } };

export type AdminCreateBoostyCredsMutationVariables = Exact<{
  input: CreateBoostyCredentialsInput;
}>;


export type AdminCreateBoostyCredsMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'CreateBoostyCredentialsPayload', boostyCredentials: { __typename?: 'AdminBoostyCredentials', id: any } } | { __typename: 'ErrorPayload', message: string } } };

export type AdminBoostyCredentialsByIdQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminBoostyCredentialsByIdQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', boostyCredentials?: { __typename?: 'AdminBoostyCredentials', createdAt: any, deviceId: string, blogName: string, state: BoostyCredentialsStateEnum, createdBy: { __typename?: 'AdminUser', email?: string | null }, tiers: { __typename?: 'AdminBoostyTiersConnection', nodes: Array<{ __typename?: 'AdminBoostyTier', id: any, name: string, subgraphs: Array<{ __typename?: 'AdminSubgraph', id: any }> }> }, members: { __typename?: 'AdminBoostyMembersConnection', nodes: Array<{ __typename?: 'AdminBoostyMember', email: string, status: string }> } } | null } };

export type AdminBoostycredentialsShowSubgraphsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminBoostycredentialsShowSubgraphsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename?: 'AdminSubgraph', id: any, name: string }> } } };

export type AdminBoostycredentialsShowSubgraphsSaveMutationVariables = Exact<{
  input: SetBoostyTierSubgraphsInput;
}>;


export type AdminBoostycredentialsShowSubgraphsSaveMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename: 'ErrorPayload', message: string } | { __typename: 'SetBoostyTierSubgraphsPayload', success: boolean } } };

export type AdminConfigVersionsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminConfigVersionsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allConfigVersions: { __typename?: 'AdminConfigVersionsConnection', nodes: Array<{ __typename?: 'AdminConfigVersion', id: any, createdAt: any, showDraftVersions: boolean, defaultLayout: string, timezone: string, createdBy: { __typename?: 'AdminUser', email?: string | null } }> } } };

export type AdminCreateConfigLatestConfigQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminCreateConfigLatestConfigQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', latestConfig: { __typename?: 'AdminConfigVersion', showDraftVersions: boolean, defaultLayout: string, timezone: string } } };

export type AdminCreateConfigVersionMutationVariables = Exact<{
  input: CreateConfigVersionInput;
}>;


export type AdminCreateConfigVersionMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename: 'CreateConfigVersionPayload', configVersion: { __typename?: 'AdminConfigVersion', id: any } } | { __typename: 'ErrorPayload', message: string } } };

export type AdminRunCronJobMutationVariables = Exact<{
  input: RunCronJobInput;
}>;


export type AdminRunCronJobMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', runCronJob: { __typename: 'ErrorPayload', message: string } | { __typename: 'RunCronJobPayload', execution: { __typename?: 'AdminCronJobExecution', id: any, job: { __typename?: 'AdminCronJob', id: any } } } } };

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


export type AdminUpdateCronJobMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', updateCronJob: { __typename: 'ErrorPayload', message: string } | { __typename: 'UpdateCronJobPayload', cronJob: { __typename?: 'AdminCronJob', id: any, expression: string, enabled: boolean } } } };

export type AdminBuildInfoQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminBuildInfoQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', buildGitCommit: string } };

export type AdminRecentlyModifiedNotesQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminRecentlyModifiedNotesQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', recentlyModifiedNoteViews: Array<{ __typename?: 'NoteView', id: string, title: string, permalink: string }> } };

export type DisableGitTokenMutationVariables = Exact<{
  input: DisableGitTokenInput;
}>;


export type DisableGitTokenMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename: 'DisableGitTokenPayload', gitToken: { __typename?: 'AdminGitToken', id: any } } | { __typename: 'ErrorPayload', message: string } } };

export type AdminGitTokensQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminGitTokensQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allGitTokens: { __typename?: 'AdminGitTokensConnection', nodes: Array<{ __typename?: 'AdminGitToken', id: any, createdAt: any, description: string, canPull: boolean, canPush: boolean, disabledAt?: any | null, createdBy: { __typename?: 'AdminUser', id: any, email?: string | null }, disabledBy?: { __typename?: 'AdminUser', id: any, email?: string | null } | null }> } } };

export type AdminCreateGitTokenMutationVariables = Exact<{
  input: CreateGitTokenInput;
}>;


export type AdminCreateGitTokenMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename: 'CreateGitTokenPayload', value: string, gitToken: { __typename?: 'AdminGitToken', id: any } } | { __typename: 'ErrorPayload', message: string } } };

export type AdminHealthChecksQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminHealthChecksQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', healthChecks: Array<{ __typename?: 'HealchCheck', id: string, status: HealthCheckStatus, description: string }> } };

export type AdminDeleteHtmlInjectionMutationVariables = Exact<{
  input: DeleteHtmlInjectionInput;
}>;


export type AdminDeleteHtmlInjectionMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename: 'DeleteHtmlInjectionPayload', deletedId: any } | { __typename: 'ErrorPayload', message: string } } };

export type AdminHtmlInjectionsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminHtmlInjectionsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allHtmlInjections: { __typename?: 'AdminHtmlInjectionsConnection', nodes: Array<{ __typename?: 'AdminHtmlInjection', id: any, createdAt: any, activeFrom?: any | null, activeTo?: any | null, description: string, position: number, placement: string }> } } };

export type AdminCreateHtmlInjectionMutationMutationVariables = Exact<{
  input: CreateHtmlInjectionInput;
}>;


export type AdminCreateHtmlInjectionMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename: 'CreateHtmlInjectionPayload', htmlInjection: { __typename?: 'AdminHtmlInjection', id: any } } | { __typename: 'ErrorPayload', message: string } } };

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


export type AdminUpdateHtmlInjectionMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename: 'ErrorPayload', message: string } | { __typename: 'UpdateHtmlInjectionPayload', htmlInjection: { __typename?: 'AdminHtmlInjection', id: any } } } };

export type AdminNoteAssetsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminNoteAssetsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allLatestNoteAssets: { __typename?: 'AdminLatestNoteAssetsConnection', nodes: Array<{ __typename?: 'AdminNoteAsset', id: any, absolutePath: string, fileName: string, size: any }> } } };

export type AdminNoteAssetQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminNoteAssetQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', noteAsset?: { __typename?: 'AdminNoteAsset', id: any, absolutePath: string, fileName: string, size: any, createdAt: any, url: string } | null } };

export type AdminListNoteViewsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminListNoteViewsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allLatestNoteViews: { __typename?: 'AdminLatestNoteViewsConnection', nodes: Array<{ __typename?: 'NoteView', id: string, path: string, title: string, free: boolean, permalink: string }> } } };

export type AdminGraphQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminGraphQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename?: 'AdminSubgraph', name: string, color?: string | null }> }, allLatestNoteViews: { __typename?: 'AdminLatestNoteViewsConnection', nodes: Array<{ __typename?: 'NoteView', id: string, subgraphNames: Array<string>, title: string, pathId: any, free: boolean, isHomePage: boolean, graphPosition?: { __typename?: 'Vector2', x: number, y: number } | null, inLinks: Array<{ __typename?: 'NoteView', title: string, pathId: any, id: string }> }> } } };

export type UpdateNoteGraphPositionsMutationVariables = Exact<{
  input: UpdateNoteGraphPositionsInput;
}>;


export type UpdateNoteGraphPositionsMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'ErrorPayload', message: string } | { __typename: 'UpdateNoteGraphPositionsPayload', success: boolean } } };

export type AdminSelectNoteViewQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminSelectNoteViewQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allLatestNoteViews: { __typename?: 'AdminLatestNoteViewsConnection', nodes: Array<{ __typename?: 'NoteView', versionId: any, path: string, title: string }> } } };

export type AdminNoteViewQueryVariables = Exact<{
  id: Scalars['String']['input'];
}>;


export type AdminNoteViewQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', noteView?: { __typename: 'NoteView', path: string, title: string, permalink: string, content: string } | null } };

export type AdminNoteWarningsQueryVariables = Exact<{
  filter?: InputMaybe<AdminLatestNoteViewsFilter>;
}>;


export type AdminNoteWarningsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allLatestNoteViews: { __typename?: 'AdminLatestNoteViewsConnection', nodes: Array<{ __typename?: 'NoteView', id: string, path: string, warnings: Array<{ __typename?: 'NoteWarning', level: NoteWarningLevelEnum, message: string }> }> } } };

export type AdminResetNotFoundPathMutationVariables = Exact<{
  input: ResetNotFoundPathInput;
}>;


export type AdminResetNotFoundPathMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename: 'ErrorPayload', message: string } | { __typename: 'ResetNotFoundPathPayload', notFoundPath: { __typename?: 'AdminNotFoundPath', id: any } } } };

export type AdminNotFoundPathsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminNotFoundPathsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allNotFoundPaths: { __typename?: 'AdminNotFoundPathsConnection', nodes: Array<{ __typename?: 'AdminNotFoundPath', id: any, path: string, totalHits: any, lastHitAt: any }> } } };

export type AdminShowNotFoundPathQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminShowNotFoundPathQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allNotFoundPaths: { __typename?: 'AdminNotFoundPathsConnection', nodes: Array<{ __typename?: 'AdminNotFoundPath', id: any, path: string, totalHits: any, lastHitAt: any }> } } };

export type AdminDeleteNotFoundIgnoredPatternMutationVariables = Exact<{
  input: DeleteNotFoundIgnoredPatternInput;
}>;


export type AdminDeleteNotFoundIgnoredPatternMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'DeleteNotFoundIgnoredPatternPayload', deletedId: any } | { __typename: 'ErrorPayload', message: string } } };

export type AdminNotFoundIgnoredPatternsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminNotFoundIgnoredPatternsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allNotFoundIgnoredPatterns: { __typename?: 'AdminNotFoundIgnoredPatternsConnection', nodes: Array<{ __typename?: 'AdminNotFoundIgnoredPattern', id: any, pattern: string, createdAt: any, createdBy: { __typename?: 'AdminUser', id: any, email?: string | null } }> } } };

export type AdminCreateNotFoundIgnoredPatternMutationMutationVariables = Exact<{
  input: CreateNotFoundIgnoredPatternInput;
}>;


export type AdminCreateNotFoundIgnoredPatternMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'CreateNotFoundIgnoredPatternPayload', notFoundIgnoredPattern: { __typename?: 'AdminNotFoundIgnoredPattern', id: any } } | { __typename: 'ErrorPayload', message: string } } };

export type AdminShowNotFoundIgnoredPatternQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminShowNotFoundIgnoredPatternQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allNotFoundIgnoredPatterns: { __typename?: 'AdminNotFoundIgnoredPatternsConnection', nodes: Array<{ __typename?: 'AdminNotFoundIgnoredPattern', id: any, pattern: string, createdAt: any, createdBy: { __typename?: 'AdminUser', id: any, email?: string | null } }> } } };

export type AdminDeleteNotFoundIgnoredPatternMutationMutationVariables = Exact<{
  input: DeleteNotFoundIgnoredPatternInput;
}>;


export type AdminDeleteNotFoundIgnoredPatternMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename: 'DeleteNotFoundIgnoredPatternPayload', deletedId: any } | { __typename: 'ErrorPayload', message: string } } };

export type AdminUpdateNotFoundIgnoredPatternMutationMutationVariables = Exact<{
  input: UpdateNotFoundIgnoredPatternInput;
}>;


export type AdminUpdateNotFoundIgnoredPatternMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename: 'ErrorPayload', message: string } | { __typename: 'UpdateNotFoundIgnoredPatternPayload', notFoundIgnoredPattern: { __typename?: 'AdminNotFoundIgnoredPattern', id: any } } } };

export type AdminOffersQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminOffersQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allOffers: { __typename?: 'AdminOffersConnection', nodes: Array<{ __typename?: 'AdminOffer', id: any, publicId: string, createdAt: any, lifetime?: string | null, priceUSD: number, startsAt?: any | null, endsAt?: any | null, subgraphs: Array<{ __typename?: 'AdminSubgraph', name: string }> }> } } };

export type AdminCreateOfferMutationMutationVariables = Exact<{
  input: CreateOfferInput;
}>;


export type AdminCreateOfferMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'CreateOfferPayload', offer: { __typename?: 'AdminOffer', id: any, publicId: string } } | { __typename: 'ErrorPayload', message: string } } };

export type AdminShowOfferQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminShowOfferQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', offer?: { __typename?: 'AdminOffer', id: any, publicId: string, createdAt: any, lifetime?: string | null, priceUSD: number, startsAt?: any | null, endsAt?: any | null, subgraphIds: Array<any>, subgraphs: Array<{ __typename?: 'AdminSubgraph', id: any, name: string }> } | null } };

export type AdminUpdateOfferMutationMutationVariables = Exact<{
  input: UpdateOfferInput;
}>;


export type AdminUpdateOfferMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'ErrorPayload', message: string } | { __typename: 'UpdateOfferPayload', offer: { __typename?: 'AdminOffer', id: any, publicId: string } } } };

export type AdminDeletePatreonCredentialsMutationVariables = Exact<{
  input: DeletePatreonCredentialsInput;
}>;


export type AdminDeletePatreonCredentialsMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'DeletePatreonCredentialsPayload', patreonCredentials: { __typename?: 'AdminPatreonCredentials', id: any } } | { __typename: 'ErrorPayload', message: string } } };

export type RefreshPatreonDataMutationVariables = Exact<{
  input: RefreshPatreonDataInput;
}>;


export type RefreshPatreonDataMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'ErrorPayload', message: string } | { __typename: 'RefreshPatreonDataPayload', success: boolean } } };

export type AdminRestorePatreonCredentialsMutationVariables = Exact<{
  input: RestorePatreonCredentialsInput;
}>;


export type AdminRestorePatreonCredentialsMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'ErrorPayload', message: string } | { __typename: 'RestorePatreonCredentialsPayload', patreonCredentials: { __typename?: 'AdminPatreonCredentials', id: any } } } };

export type AdminPatreonCredentialsQueryVariables = Exact<{
  filter?: InputMaybe<AdminPatreonCredentialsFilterInput>;
}>;


export type AdminPatreonCredentialsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allPatreonCredentials: { __typename?: 'AdminPatreonCredentialsConnection', nodes: Array<{ __typename?: 'AdminPatreonCredentials', id: any, state: PatreonCredentialsStateEnum, creatorAccessToken: string, createdAt: any, syncedAt?: any | null, createdBy: { __typename?: 'AdminUser', id: any, email?: string | null } }> } } };

export type AdminCreatePatreonCredsMutationVariables = Exact<{
  input: CreatePatreonCredentialsInput;
}>;


export type AdminCreatePatreonCredsMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'CreatePatreonCredentialsPayload', patreonCredentials: { __typename?: 'AdminPatreonCredentials', id: any } } | { __typename: 'ErrorPayload', message: string } } };

export type AdminPatreonCredentialsByIdQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminPatreonCredentialsByIdQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', patreonCredentials?: { __typename?: 'AdminPatreonCredentials', createdAt: any, creatorAccessToken: string, state: PatreonCredentialsStateEnum, createdBy: { __typename?: 'AdminUser', email?: string | null }, tiers: { __typename?: 'AdminPatreonTiersConnection', nodes: Array<{ __typename?: 'AdminPatreonTier', id: any, missedAt?: any | null, title: string, amountCents: any, subgraphs: Array<{ __typename?: 'AdminSubgraph', id: any }> }> }, members: { __typename?: 'AdminPatreonMembersConnection', nodes: Array<{ __typename?: 'AdminPatreonMember', email: string, status: string, currentTier?: { __typename?: 'AdminPatreonTier', title: string } | null }> } } | null } };

export type AdminPatreoncredentialsShowSubgraphsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminPatreoncredentialsShowSubgraphsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename?: 'AdminSubgraph', id: any, name: string }> } } };

export type AdminPatreoncredentialsShowSubgraphsSaveMutationVariables = Exact<{
  input: SetPatreonTierSubgraphsInput;
}>;


export type AdminPatreoncredentialsShowSubgraphsSaveMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename: 'ErrorPayload', message: string } | { __typename: 'SetPatreonTierSubgraphsPayload', success: boolean } } };

export type AdminPurchasesQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminPurchasesQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allPurchases: { __typename?: 'AdminPurchasesConnection', nodes: Array<{ __typename?: 'AdminPurchase', id: string, createdAt: any, paymentProvider: string, status: string, successful: boolean, offerId: any, email: string }> } } };

export type AdminRedirectsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminRedirectsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allRedirects: { __typename?: 'AdminRedirectsConnection', nodes: Array<{ __typename?: 'AdminRedirect', id: any, createdAt: any, pattern: string, ignoreCase: boolean, isRegex: boolean, target: string, createdBy: { __typename?: 'AdminUser', id: any, email?: string | null } }> } } };

export type AdminCreateRedirectMutationMutationVariables = Exact<{
  input: CreateRedirectInput;
}>;


export type AdminCreateRedirectMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'CreateRedirectPayload', redirect: { __typename?: 'AdminRedirect', id: any } } | { __typename: 'ErrorPayload', message: string } } };

export type AdminShowRedirectQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminShowRedirectQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', redirect?: { __typename?: 'AdminRedirect', id: any, createdAt: any, pattern: string, ignoreCase: boolean, isRegex: boolean, target: string, createdBy: { __typename?: 'AdminUser', id: any, email?: string | null } } | null } };

export type AdminDeleteRedirectMutationMutationVariables = Exact<{
  input: DeleteRedirectInput;
}>;


export type AdminDeleteRedirectMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'DeleteRedirectPayload', id: any } | { __typename: 'ErrorPayload', message: string } } };

export type AdminUpdateRedirectMutationMutationVariables = Exact<{
  input: UpdateRedirectInput;
}>;


export type AdminUpdateRedirectMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'ErrorPayload', message: string } | { __typename: 'UpdateRedirectPayload', redirect: { __typename?: 'AdminRedirect', id: any } } } };

export type AdminMakeReleaseLiveMutationVariables = Exact<{
  input: MakeReleaseLiveInput;
}>;


export type AdminMakeReleaseLiveMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'ErrorPayload', message: string } | { __typename: 'MakeReleaseLivePayload', release: { __typename?: 'AdminRelease', id: any } } } };

export type AdminReleasesQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminReleasesQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allReleases: { __typename?: 'AdminReleasesConnection', nodes: Array<{ __typename?: 'AdminRelease', id: any, createdAt: any, title: string, isLive: boolean, createdBy: { __typename?: 'AdminUser', email?: string | null } }> } } };

export type AdminCreateReleaseMutationVariables = Exact<{
  input: CreateReleaseInput;
}>;


export type AdminCreateReleaseMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'CreateReleasePayload', release: { __typename?: 'AdminRelease', id: any } } | { __typename: 'ErrorPayload', message: string } } };

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


export type UpdateSubgraphMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'ErrorPayload', message: string } | { __typename: 'UpdateSubgraphPayload', subgraph: { __typename?: 'AdminSubgraph', id: any, color?: string | null } } } };

export type AdminTelegramAccountsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminTelegramAccountsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allTelegramAccounts: { __typename?: 'AdminTelegramAccountsConnection', nodes: Array<{ __typename?: 'AdminTelegramAccount', id: any, phone: string, displayName: string, isPremium: boolean, enabled: boolean, createdAt: any, createdBy?: { __typename?: 'User', email?: string | null } | null }> } } };

export type StartTelegramAccountAuthMutationVariables = Exact<{
  input: AdminStartTelegramAccountAuthInput;
}>;


export type StartTelegramAccountAuthMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'AdminStartTelegramAccountAuthPayload', authState: { __typename?: 'AdminTelegramAccountAuthState', phone: string, state: AdminTelegramAccountAuthStateEnum, passwordHint?: string | null } } | { __typename: 'ErrorPayload', message: string } } };

export type CompleteTelegramAccountAuthMutationVariables = Exact<{
  input: AdminCompleteTelegramAccountAuthInput;
}>;


export type CompleteTelegramAccountAuthMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'AdminCompleteTelegramAccountAuthPayload', account: { __typename?: 'AdminTelegramAccount', id: any, phone: string, displayName: string, isPremium: boolean } } | { __typename: 'ErrorPayload', message: string } } };

export type AdminTelegramAccountDialogsQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminTelegramAccountDialogsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', telegramAccount?: { __typename?: 'AdminTelegramAccount', dialogs: Array<{ __typename?: 'AdminTelegramAccountDialog', id: any, username: string, title: string, type: AdminTelegramAccountDialogType, publishTags: Array<{ __typename?: 'AdminTelegramPublishTag', id: any }>, publishInstantTags: Array<{ __typename?: 'AdminTelegramPublishTag', id: any }> }> } | null } };

export type AdminImportTelegramAccountChannelMutationVariables = Exact<{
  input: AdminImportTelegramAccountChannelInput;
}>;


export type AdminImportTelegramAccountChannelMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'AdminImportTelegramAccountChannelPayload', success: boolean } | { __typename: 'ErrorPayload', message: string } } };

export type AdminTelegramAccountShowDialogsInstantTagsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminTelegramAccountShowDialogsInstantTagsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allTelegramPublishTags: { __typename?: 'AdminTelegramPublishTagsConnection', nodes: Array<{ __typename?: 'AdminTelegramPublishTag', id: any, label: string }> } } };

export type AdminTelegramAccountShowDialogsInstantTagsSaveMutationVariables = Exact<{
  input: AdminSetTelegramAccountChatPublishInstantTagsInput;
}>;


export type AdminTelegramAccountShowDialogsInstantTagsSaveMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename: 'AdminSetTelegramAccountChatPublishInstantTagsPayload', success: boolean } | { __typename: 'ErrorPayload', message: string } } };

export type AdminTelegramAccountShowDialogsTagsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminTelegramAccountShowDialogsTagsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allTelegramPublishTags: { __typename?: 'AdminTelegramPublishTagsConnection', nodes: Array<{ __typename?: 'AdminTelegramPublishTag', id: any, label: string }> } } };

export type AdminTelegramAccountShowDialogsTagsSaveMutationVariables = Exact<{
  input: AdminSetTelegramAccountChatPublishTagsInput;
}>;


export type AdminTelegramAccountShowDialogsTagsSaveMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'AdminSetTelegramAccountChatPublishTagsPayload', success: boolean } | { __typename: 'ErrorPayload', message: string } } };

export type AdminSignOutTelegramAccountMutationVariables = Exact<{
  input: AdminSignOutTelegramAccountInput;
}>;


export type AdminSignOutTelegramAccountMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename: 'AdminSignOutTelegramAccountPayload', success: boolean } | { __typename?: 'ErrorPayload', message: string } } };

export type AdminTelegramAccountUpdateQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminTelegramAccountUpdateQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', telegramAccount?: { __typename?: 'AdminTelegramAccount', id: any, phone: string, displayName: string, isPremium: boolean, enabled: boolean } | null } };

export type AdminUpdateTelegramAccountMutationMutationVariables = Exact<{
  input: AdminUpdateTelegramAccountInput;
}>;


export type AdminUpdateTelegramAccountMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'AdminUpdateTelegramAccountPayload', account: { __typename?: 'AdminTelegramAccount', id: any, enabled: boolean } } | { __typename: 'ErrorPayload', message: string } } };

export type AdminResetTelegramPublishNoteMutationVariables = Exact<{
  input: ResetTelegramPublishNoteInput;
}>;


export type AdminResetTelegramPublishNoteMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'ErrorPayload', message: string } | { __typename: 'ResetTelegramPublishNotePayload', publishNote: { __typename?: 'AdminTelegramPublishNote', id: any } } } };

export type AdminSendTelegramPublishNoteNowMutationVariables = Exact<{
  input: SendTelegramPublishNoteNowInput;
}>;


export type AdminSendTelegramPublishNoteNowMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'ErrorPayload', message: string } | { __typename: 'SendTelegramPublishNoteNowPayload', publishNote: { __typename?: 'AdminTelegramPublishNote', id: any } } } };

export type AdminTelegramPublishNoteCountQueryVariables = Exact<{
  filter: AdminTelegramPublishNotesFilter;
}>;


export type AdminTelegramPublishNoteCountQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allTelegramPublishNotes: { __typename?: 'AdminTelegramPublishNotesConnection', count: any } } };

export type AdminTelegramPublishNotesQueryVariables = Exact<{
  filter: AdminTelegramPublishNotesFilter;
}>;


export type AdminTelegramPublishNotesQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allTelegramPublishNotes: { __typename?: 'AdminTelegramPublishNotesConnection', nodes: Array<{ __typename?: 'AdminTelegramPublishNote', id: any, publishAt: any, secondsUntilPublish: any, publishedAt?: any | null, status: string, errorCount: any, noteView: { __typename?: 'NoteView', title: string } }> } } };

export type AdminTelegramPublishNoteQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminTelegramPublishNoteQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', telegramPublishNote?: { __typename?: 'AdminTelegramPublishNote', id: any, createdAt: any, publishAt: any, secondsUntilPublish: any, publishedAt?: any | null, status: string, tags: Array<{ __typename?: 'AdminTelegramPublishTag', label: string }>, chats: Array<{ __typename?: 'AdminTgBotChat', chatTitle: string, chatType: string }>, noteView: { __typename?: 'NoteView', title: string }, post: { __typename?: 'TelegramPost', content: string, warnings: Array<string> } } | null } };

export type AdminTgBotsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminTgBotsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allTgBots: { __typename?: 'AdminTgBotsConnection', nodes: Array<{ __typename?: 'AdminTgBot', id: any, name: string, description: string, enabled: boolean, createdAt: any, createdBy: { __typename?: 'AdminUser', email?: string | null } }> } } };

export type AdminCreateTgBotMutationMutationVariables = Exact<{
  input: CreateTgBotInput;
}>;


export type AdminCreateTgBotMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'CreateTgBotPayload', tgBot: { __typename?: 'AdminTgBot', id: any, name: string } } | { __typename: 'ErrorPayload', message: string } } };

export type AdminTgBotChatsQueryVariables = Exact<{
  filter: AdminTgBotChatsFilterInput;
}>;


export type AdminTgBotChatsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', tgBotChats: { __typename?: 'AdminTgBotChatsConnection', nodes: Array<{ __typename?: 'AdminTgBotChat', id: any, chatType: string, chatTitle: string, addedAt: any, removedAt?: any | null, memberCount: number, subgraphAccesses: Array<{ __typename?: 'AdminTgChatSubgraphAccess', id: any, subgraphId: any }> }> } } };

export type AdminTgbotShowChatsSubgraphsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminTgbotShowChatsSubgraphsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename?: 'AdminSubgraph', id: any, name: string }> } } };

export type AdminTgbotsShowchatsSubgraphsSaveMutationVariables = Exact<{
  input: SetTgChatSubgraphsInput;
}>;


export type AdminTgbotsShowchatsSubgraphsSaveMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'ErrorPayload', message: string } | { __typename: 'SetTgChatSubgraphsPayload', success: boolean } } };

export type AdminTgBotInviteChatsQueryVariables = Exact<{
  filter: AdminTgBotChatsFilterInput;
}>;


export type AdminTgBotInviteChatsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', tgBotChats: { __typename?: 'AdminTgBotChatsConnection', nodes: Array<{ __typename?: 'AdminTgBotChat', id: any, chatType: string, chatTitle: string, addedAt: any, removedAt?: any | null, memberCount: number, subgraphInvites: Array<{ __typename?: 'AdminTgBotChatSubgraphInvite', id: string, subgraphId: any }> }> } } };

export type AdminTgbotShowInviteChatsSubgraphsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminTgbotShowInviteChatsSubgraphsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allSubgraphs: { __typename?: 'AdminSubgraphsConnection', nodes: Array<{ __typename?: 'AdminSubgraph', id: any, name: string }> } } };

export type AdminTgbotShowInviteChatsSubgraphsSaveMutationVariables = Exact<{
  input: SetTgChatSubgraphInvitesInput;
}>;


export type AdminTgbotShowInviteChatsSubgraphsSaveMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'ErrorPayload', message: string } | { __typename: 'SetTgChatSubgraphInvitesPayload', success: boolean } } };

export type AdminTgbotShowPublishInstantTagsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminTgbotShowPublishInstantTagsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allTelegramPublishTags: { __typename?: 'AdminTelegramPublishTagsConnection', nodes: Array<{ __typename?: 'AdminTelegramPublishTag', id: any, label: string }> } } };

export type AdminTgbotShowPublishInstantTagsSaveMutationVariables = Exact<{
  input: SetTgChatPublishInstantTagsInput;
}>;


export type AdminTgbotShowPublishInstantTagsSaveMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', data: { __typename: 'ErrorPayload', message: string } | { __typename: 'SetTgChatPublishInstantTagsPayload', success: boolean } } };

export type AdminTgBotPublishTagsQueryVariables = Exact<{
  filter: AdminTgBotChatsFilterInput;
}>;


export type AdminTgBotPublishTagsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', tgBotChats: { __typename?: 'AdminTgBotChatsConnection', nodes: Array<{ __typename?: 'AdminTgBotChat', id: any, chatType: string, chatTitle: string, addedAt: any, removedAt?: any | null, memberCount: number, publishTags: Array<{ __typename?: 'AdminTelegramPublishTag', id: any }>, publishInstantTags: Array<{ __typename?: 'AdminTelegramPublishTag', id: any }> }> } } };

export type AdminTgbotShowPublishTagsQueryVariables = Exact<{ [key: string]: never; }>;


export type AdminTgbotShowPublishTagsQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', allTelegramPublishTags: { __typename?: 'AdminTelegramPublishTagsConnection', nodes: Array<{ __typename?: 'AdminTelegramPublishTag', id: any, label: string }> } } };

export type AdminTgbotShowPublishTagsSaveMutationVariables = Exact<{
  input: SetTgChatPublishTagsInput;
}>;


export type AdminTgbotShowPublishTagsSaveMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'ErrorPayload', message: string } | { __typename: 'SetTgChatPublishTagsPayload', success: boolean } } };

export type AdminShowTgBotQueryVariables = Exact<{
  id: Scalars['Int64']['input'];
}>;


export type AdminShowTgBotQuery = { __typename?: 'Query', admin: { __typename?: 'AdminQuery', tgBot?: { __typename?: 'AdminTgBot', id: any, name: string, description: string, enabled: boolean, createdAt: any, createdBy: { __typename?: 'AdminUser', email?: string | null } } | null } };

export type AdminUpdateTgBotMutationMutationVariables = Exact<{
  input: UpdateTgBotInput;
}>;


export type AdminUpdateTgBotMutationMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', payload: { __typename: 'ErrorPayload', message: string } | { __typename: 'UpdateTgBotPayload', tgBot: { __typename?: 'AdminTgBot', id: any, description: string } } } };

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


export type AdminCreateUserMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', createUser: { __typename: 'CreateUserPayload', user: { __typename?: 'AdminUser', id: any } } | { __typename: 'ErrorPayload', message: string } } };

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


export type AdminUpdateUserMutation = { __typename?: 'Mutation', admin: { __typename?: 'AdminMutation', updateUser: { __typename: 'ErrorPayload', message: string } | { __typename: 'UpdateUserPayload', user: { __typename?: 'AdminUser', id: any, email?: string | null } } } };

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


export type ToggleFavoriteNoteMutation = { __typename?: 'Mutation', payload: { __typename: 'ErrorPayload', message: string } | { __typename: 'ToggleFavoriteNotePayload', favoriteNotes: Array<{ __typename?: 'PublicNote', pathId: any }> } };

export type PaywallActivePurchaseQueryQueryVariables = Exact<{ [key: string]: never; }>;


export type PaywallActivePurchaseQueryQuery = { __typename?: 'Query', viewer: { __typename?: 'Viewer', activePurchases: Array<{ __typename?: 'Purchase', id: string, status: string, successful: boolean }> } };

export type CreateEmailWaitListRequestMutationMutationVariables = Exact<{
  input: CreateEmailWaitListRequestInput;
}>;


export type CreateEmailWaitListRequestMutationMutation = { __typename?: 'Mutation', createEmailWaitListRequest: { __typename: 'CreateEmailWaitListRequestPayload', success: boolean } | { __typename: 'ErrorPayload', message: string } };

export type PaywallQueryQueryVariables = Exact<{
  filter: ViewerOffersFilter;
}>;


export type PaywallQueryQuery = { __typename?: 'Query', viewer: { __typename?: 'Viewer', offers?: { __typename: 'ActiveOffers', nodes: Array<{ __typename?: 'Offer', id: string, priceUSD: number, subgraphs: Array<{ __typename?: 'Subgraph', name: string }> }> } | { __typename: 'SubgraphWaitList', tgBotUrl?: string | null, emailAllowed: boolean } | null } };

export type CreatePaymentLinkMutationVariables = Exact<{
  input: CreatePaymentLinkInput;
}>;


export type CreatePaymentLinkMutation = { __typename?: 'Mutation', data: { __typename: 'CreatePaymentLinkPayload', redirectUrl: string } | { __typename: 'ErrorPayload', message: string } };

export type SiteSearchQueryVariables = Exact<{
  input: SearchInput;
}>;


export type SiteSearchQuery = { __typename?: 'Query', search: { __typename?: 'SearchConnection', nodes: Array<{ __typename?: 'SearchResult', highlightedTitle?: string | null, highlightedContent: Array<string>, id: string }> } };

export type UserSubscriptionsQueryVariables = Exact<{ [key: string]: never; }>;


export type UserSubscriptionsQuery = { __typename?: 'Query', viewer: { __typename?: 'Viewer', user?: { __typename?: 'User', subgraphAccesses: Array<{ __typename?: 'UserSubgraphAccess', id: string, createdAt: any, expiresAt?: any | null, subgraph: { __typename?: 'Subgraph', name: string, homePath: string } }> } | null } };

export function $trip2g_graphql_request(query: '\n\t\tquery Admins {\n\t\t\tadmin {\n\t\t\t\tallAdmins {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tgrantedAt\n\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation DisableApiKey($input: DisableApiKeyInput!) {\n\t\t\tadmin {\n\t\t\t\tdata: disableApiKey(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on DisableApiKeyPayload {\n\t\t\t\t\t\tapiKey {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: DisableApiKeyMutationVariables) => DisableApiKeyMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminListApiKeys {\n\t\t\tadmin {\n\t\t\t\tallApiKeys {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tdescription\n\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t\tdisabledAt\n\t\t\t\t\t\tdisabledBy {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminListApiKeysQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminCreateApiKey($input: CreateApiKeyInput!) {\n\t\t\tadmin {\n\t\t\t\tdata: createApiKey(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on CreateApiKeyPayload {\n\t\t\t\t\t\tvalue\n\t\t\t\t\t\tapiKey {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCreateApiKeyMutationVariables) => AdminCreateApiKeyMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminApiKeyShowQuery($filter: ApiKeyLogsFilterInput!) {\n\t\t\tadmin {\n\t\t\t\tapiKeyLogs(filter: $filter) {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tactionName\n\t\t\t\t\t\tip\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminApiKeyShowQueryQueryVariables) => AdminApiKeyShowQueryQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminAuditLogs($filter: AdminAuditLogsFilterInput!) {\n\t\t\tadmin {\n\t\t\t\tauditLogs(filter: $filter) {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tlevel\n\t\t\t\t\t\tmessage\n\t\t\t\t\t\tparams\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminAuditLogsQueryVariables) => AdminAuditLogsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminClearBackgroundQueue($input: ClearBackgroundQueueInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: clearBackgroundQueue(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on ClearBackgroundQueuePayload {\n\t\t\t\t\t\tqueue {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tstopped\n\t\t\t\t\t\t}\n\t\t\t\t\t\tdeletedCount\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminClearBackgroundQueueMutationVariables) => AdminClearBackgroundQueueMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminStartBackgroundQueue($input: StartBackgroundQueueInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: startBackgroundQueue(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on StartBackgroundQueuePayload {\n\t\t\t\t\t\tqueues {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tstopped\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminStartBackgroundQueueMutationVariables) => AdminStartBackgroundQueueMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminStopBackgroundQueue($input: StopBackgroundQueueInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: stopBackgroundQueue(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on StopBackgroundQueuePayload {\n\t\t\t\t\t\tqueues {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tstopped\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminStopBackgroundQueueMutationVariables) => AdminStopBackgroundQueueMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminBackgroundQueues {\n\t\t\tadmin {\n\t\t\t\tallBackgroundQueues {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tpendingCount\n\t\t\t\t\t\tretryCount\n\t\t\t\t\t\tstopped\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminBackgroundQueuesQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminBackgroundQueue($id: String!) {\n\t\t\tadmin {\n\t\t\t\tbackgroundQueue(id: $id) {\n\t\t\t\t\tid\n\t\t\t\t\tpendingCount\n\t\t\t\t\tretryCount\n\t\t\t\t\tstopped\n\t\t\t\t\tjobs @exportType(name: "Job", single: true) {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tname\n\t\t\t\t\t\tparams\n\t\t\t\t\t\tretryCount\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminBackgroundQueueQueryVariables) => AdminBackgroundQueueQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminDeleteBoostyCredentials($input: DeleteBoostyCredentialsInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: deleteBoostyCredentials(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on ErrorPayload{\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on DeleteBoostyCredentialsPayload {\n\t\t\t\t\t\tboostyCredentials {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminDeleteBoostyCredentialsMutationVariables) => AdminDeleteBoostyCredentialsMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation RefreshBoostyData($input: RefreshBoostyDataInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: refreshBoostyData(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on RefreshBoostyDataPayload {\n\t\t\t\t\t\tsuccess\n\t\t\t\t\t\tcredentials {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: RefreshBoostyDataMutationVariables) => RefreshBoostyDataMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminRestoreBoostyCredentials($input: RestoreBoostyCredentialsInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: restoreBoostyCredentials(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on RestoreBoostyCredentialsPayload {\n\t\t\t\t\t\tboostyCredentials {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminRestoreBoostyCredentialsMutationVariables) => AdminRestoreBoostyCredentialsMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminBoostyCredentials($filter: AdminBoostyCredentialsFilterInput) {\n\t\t\tadmin {\n\t\t\t\tallBoostyCredentials(filter: $filter) {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tstate\n\t\t\t\t\t\tdeviceId\n\t\t\t\t\t\tblogName\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminBoostyCredentialsQueryVariables) => AdminBoostyCredentialsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminCreateBoostyCreds($input: CreateBoostyCredentialsInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: createBoostyCredentials(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on CreateBoostyCredentialsPayload {\n\t\t\t\t\t\tboostyCredentials {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCreateBoostyCredsMutationVariables) => AdminCreateBoostyCredsMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminBoostyCredentialsById($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\tboostyCredentials(id: $id) {\n\t\t\t\t\tcreatedAt\n\t\t\t\t\tdeviceId\n\t\t\t\t\tblogName\n\t\t\t\t\tstate\n\n\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\temail\n\t\t\t\t\t}\n\n\t\t\t\t\ttiers {\n\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tname\n\n\t\t\t\t\t\t\tsubgraphs {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t\n\t\t\t\t\tmembers {\n\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\tstatus\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminBoostyCredentialsByIdQueryVariables) => AdminBoostyCredentialsByIdQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminBoostycredentialsShowSubgraphs {\n\t\t\tadmin {\n\t\t\t\tallSubgraphs {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tname\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminBoostycredentialsShowSubgraphsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminBoostycredentialsShowSubgraphsSave($input: SetBoostyTierSubgraphsInput!) {\n\t\t\tadmin {\n\t\t\t\tdata: setBoostyTierSubgraphs(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on SetBoostyTierSubgraphsPayload {\n\t\t\t\t\t\tsuccess\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminBoostycredentialsShowSubgraphsSaveMutationVariables) => AdminBoostycredentialsShowSubgraphsSaveMutation

export function $trip2g_graphql_request(query: '\n\t\t\tquery AdminConfigVersions {\n\t\t\t\tadmin {\n\t\t\t\t\tallConfigVersions {\n\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\tshowDraftVersions\n\t\t\t\t\t\t\tdefaultLayout\n\t\t\t\t\t\t\ttimezone\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): () => AdminConfigVersionsQuery

export function $trip2g_graphql_request(query: '\n\t\t\tquery AdminCreateConfigLatestConfig {\n\t\t\t\tadmin {\n\t\t\t\t\tlatestConfig {\n\t\t\t\t\t\tshowDraftVersions\n\t\t\t\t\t\tdefaultLayout\n\t\t\t\t\t\ttimezone\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): () => AdminCreateConfigLatestConfigQuery

export function $trip2g_graphql_request(query: '\n\t\t\tmutation AdminCreateConfigVersion($input: CreateConfigVersionInput!) {\n\t\t\t\tadmin {\n\t\t\t\t\tdata: createConfigVersion(input: $input) {\n\t\t\t\t\t\t__typename\n\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t\t... on CreateConfigVersionPayload {\n\t\t\t\t\t\t\tconfigVersion {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): (variables: AdminCreateConfigVersionMutationVariables) => AdminCreateConfigVersionMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminRunCronJob($input: RunCronJobInput!) {\n\t\t\tadmin {\n\t\t\t\trunCronJob(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on RunCronJobPayload {\n\t\t\t\t\t\texecution {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tjob {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminRunCronJobMutationVariables) => AdminRunCronJobMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminAllCronJobs {\n\t\t\tadmin {\n\t\t\t\tallCronJobs {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tname\n\t\t\t\t\t\tenabled\n\t\t\t\t\t\texpression\n\t\t\t\t\t\tlastExecAt\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminAllCronJobsQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminCronJobExecutions($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\tcronJob(id: $id) {\n\t\t\t\t\tid\n\t\t\t\t\texecutions {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tstartedAt\n\t\t\t\t\t\tfinishedAt\n\t\t\t\t\t\tstatus\n\t\t\t\t\t\terrorMessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCronJobExecutionsQueryVariables) => AdminCronJobExecutionsQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminCronJobShow($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\tcronJob(id: $id) {\n\t\t\t\t\tid\n\t\t\t\t\tname\n\t\t\t\t\tenabled\n\t\t\t\t\texpression\n\t\t\t\t\tlastExecAt\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCronJobShowQueryVariables) => AdminCronJobShowQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminCronJobUpdate($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\tcronJob(id: $id) {\n\t\t\t\t\tid\n\t\t\t\t\tname\n\t\t\t\t\tenabled\n\t\t\t\t\texpression\n\t\t\t\t\tlastExecAt\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCronJobUpdateQueryVariables) => AdminCronJobUpdateQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminUpdateCronJob($input: UpdateCronJobInput!) {\n\t\t\tadmin {\n\t\t\t\tupdateCronJob(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on UpdateCronJobPayload {\n\t\t\t\t\t\tcronJob {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\texpression\n\t\t\t\t\t\t\tenabled\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminUpdateCronJobMutationVariables) => AdminUpdateCronJobMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminBuildInfo {\n\t\t\tadmin {\n\t\t\t\tbuildGitCommit\n\t\t\t}\n\t\t}\n\t'): () => AdminBuildInfoQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminRecentlyModifiedNotes {\n\t\t\tadmin {\n\t\t\t\trecentlyModifiedNoteViews {\n\t\t\t\tid\n\t\t\t\ttitle\n\t\t\t\tpermalink\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminRecentlyModifiedNotesQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation DisableGitToken($input: DisableGitTokenInput!) {\n\t\t\tadmin {\n\t\t\t\tdata: disableGitToken(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on DisableGitTokenPayload {\n\t\t\t\t\t\tgitToken {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: DisableGitTokenMutationVariables) => DisableGitTokenMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminGitTokens {\n\t\t\tadmin {\n\t\t\t\tallGitTokens {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tdescription\n\t\t\t\t\t\tcanPull\n\t\t\t\t\t\tcanPush\n\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t\tdisabledAt\n\t\t\t\t\t\tdisabledBy {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminGitTokensQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminCreateGitToken($input: CreateGitTokenInput!) {\n\t\t\tadmin {\n\t\t\t\tdata: createGitToken(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on CreateGitTokenPayload {\n\t\t\t\t\t\tvalue\n\t\t\t\t\t\tgitToken {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCreateGitTokenMutationVariables) => AdminCreateGitTokenMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminHealthChecks {\n\t\t\tadmin {\n\t\t\t\thealthChecks {\n\t\t\t\t\tid\n\t\t\t\t\tstatus\n\t\t\t\t\tdescription\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminHealthChecksQuery

export function $trip2g_graphql_request(query: '\n\t\t\tmutation AdminDeleteHtmlInjection($input: DeleteHtmlInjectionInput!) {\n\t\t\t\tadmin {\n\t\t\t\t\tdata: deleteHtmlInjection(input: $input) {\n\t\t\t\t\t\t__typename\n\t\t\t\t\t\t... on DeleteHtmlInjectionPayload {\n\t\t\t\t\t\t\tdeletedId\n\t\t\t\t\t\t}\n\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): (variables: AdminDeleteHtmlInjectionMutationVariables) => AdminDeleteHtmlInjectionMutation

export function $trip2g_graphql_request(query: '\n\t\t\tquery AdminHtmlInjections {\n\t\t\t\tadmin {\n\t\t\t\t\tallHtmlInjections {\n\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\t\tactiveFrom\n\t\t\t\t\t\t\tactiveTo\n\t\t\t\t\t\t\tdescription\n\t\t\t\t\t\t\tposition\n\t\t\t\t\t\t\tplacement\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): () => AdminHtmlInjectionsQuery

export function $trip2g_graphql_request(query: '\n\t\t\tmutation AdminCreateHtmlInjectionMutation($input: CreateHtmlInjectionInput!) {\n\t\t\t\tadmin {\n\t\t\t\t\tdata: createHtmlInjection(input: $input) {\n\t\t\t\t\t\t__typename\n\t\t\t\t\t\t... on CreateHtmlInjectionPayload {\n\t\t\t\t\t\t\thtmlInjection {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): (variables: AdminCreateHtmlInjectionMutationMutationVariables) => AdminCreateHtmlInjectionMutationMutation

export function $trip2g_graphql_request(query: '\n\t\t\tquery AdminShowHtmlInjection($id: Int64!) {\n\t\t\t\tadmin {\n\t\t\t\t\thtmlInjection(id: $id) {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tactiveFrom\n\t\t\t\t\t\tactiveTo\n\t\t\t\t\t\tdescription\n\t\t\t\t\t\tposition\n\t\t\t\t\t\tplacement\n\t\t\t\t\t\tcontent\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): (variables: AdminShowHtmlInjectionQueryVariables) => AdminShowHtmlInjectionQuery

export function $trip2g_graphql_request(query: '\n\t\t\tquery AdminUpdateDataHtmlInjection($id: Int64!) {\n\t\t\t\tadmin {\n\t\t\t\t\thtmlInjection(id: $id) {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tactiveFrom\n\t\t\t\t\t\tactiveTo\n\t\t\t\t\t\tdescription\n\t\t\t\t\t\tposition\n\t\t\t\t\t\tplacement\n\t\t\t\t\t\tcontent\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): (variables: AdminUpdateDataHtmlInjectionQueryVariables) => AdminUpdateDataHtmlInjectionQuery

export function $trip2g_graphql_request(query: '\n\t\t\tmutation AdminUpdateHtmlInjection($input: UpdateHtmlInjectionInput!) {\n\t\t\t\tadmin {\n\t\t\t\t\tdata: updateHtmlInjection(input: $input) {\n\t\t\t\t\t\t__typename\n\t\t\t\t\t\t... on UpdateHtmlInjectionPayload {\n\t\t\t\t\t\t\thtmlInjection {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): (variables: AdminUpdateHtmlInjectionMutationVariables) => AdminUpdateHtmlInjectionMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminNoteAssets {\n\t\t\tadmin {\n\t\t\t\tallLatestNoteAssets {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tabsolutePath\n\t\t\t\t\t\tfileName\n\t\t\t\t\t\tsize\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminNoteAssetsQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminNoteAsset($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\tnoteAsset(id: $id) {\n\t\t\t\t\tid\n\t\t\t\t\tabsolutePath\n\t\t\t\t\tfileName\n\t\t\t\t\tsize\n\t\t\t\t\tcreatedAt\n\t\t\t\t\turl\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminNoteAssetQueryVariables) => AdminNoteAssetQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminListNoteViews {\n\t\t\tadmin {\n\t\t\t\tallLatestNoteViews {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tpath\n\t\t\t\t\t\ttitle\n\t\t\t\t\t\tfree\n\t\t\t\t\t\tpermalink\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminListNoteViewsQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminGraph {\n\t\t\tadmin {\n\t\t\t\tallSubgraphs {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tname\n\t\t\t\t\t\tcolor\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t\tallLatestNoteViews {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tsubgraphNames\n\t\t\t\t\t\ttitle\n\t\t\t\t\t\tpathId\n\t\t\t\t\t\tfree\n\t\t\t\t\t\tisHomePage\n\t\t\t\t\t\tgraphPosition{\n\t\t\t\t\t\t\tx,\n\t\t\t\t\t\t\ty,\n\t\t\t\t\t\t}\n\t\t\t\t\t\tinLinks {\n\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t\tpathId\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n'): () => AdminGraphQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation UpdateNoteGraphPositions($input: UpdateNoteGraphPositionsInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: updateNoteGraphPositions(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on UpdateNoteGraphPositionsPayload {\n\t\t\t\t\t\tsuccess\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: UpdateNoteGraphPositionsMutationVariables) => UpdateNoteGraphPositionsMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminSelectNoteView {\n\t\t\tadmin {\n\t\t\t\tallLatestNoteViews {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tversionId\n\t\t\t\t\t\tpath\n\t\t\t\t\t\ttitle\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminSelectNoteViewQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminNoteView($id: String!) {\n\t\t\tadmin {\n\t\t\t\tnoteView(id: $id) {\n\t\t\t\t\t__typename\n\t\t\t\t\tpath\n\t\t\t\t\ttitle\n\t\t\t\t\tpermalink\n\t\t\t\t\tcontent\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminNoteViewQueryVariables) => AdminNoteViewQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminNoteWarnings($filter: AdminLatestNoteViewsFilter) {\n\t\t\tadmin {\n\t\t\t\tallLatestNoteViews(filter: $filter) {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tpath\n\t\t\t\t\t\twarnings {\n\t\t\t\t\t\t\tlevel\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminNoteWarningsQueryVariables) => AdminNoteWarningsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminResetNotFoundPath($input: ResetNotFoundPathInput!) {\n\t\t\tadmin {\n\t\t\t\tdata: resetNotFoundPath(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on ResetNotFoundPathPayload {\n\t\t\t\t\t\tnotFoundPath {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminResetNotFoundPathMutationVariables) => AdminResetNotFoundPathMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminNotFoundPaths {\n\t\t\tadmin {\n\t\t\t\tallNotFoundPaths {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tpath\n\t\t\t\t\t\ttotalHits\n\t\t\t\t\t\tlastHitAt\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminNotFoundPathsQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminShowNotFoundPath {\n\t\t\tadmin {\n\t\t\t\tallNotFoundPaths {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tpath\n\t\t\t\t\t\ttotalHits\n\t\t\t\t\t\tlastHitAt\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminShowNotFoundPathQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminDeleteNotFoundIgnoredPattern($input: DeleteNotFoundIgnoredPatternInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: deleteNotFoundIgnoredPattern(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on DeleteNotFoundIgnoredPatternPayload {\n\t\t\t\t\t\tdeletedId\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminDeleteNotFoundIgnoredPatternMutationVariables) => AdminDeleteNotFoundIgnoredPatternMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminNotFoundIgnoredPatterns {\n\t\t\tadmin {\n\t\t\t\tallNotFoundIgnoredPatterns {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tpattern\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminNotFoundIgnoredPatternsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminCreateNotFoundIgnoredPatternMutation($input: CreateNotFoundIgnoredPatternInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: createNotFoundIgnoredPattern(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on CreateNotFoundIgnoredPatternPayload {\n\t\t\t\t\t\tnotFoundIgnoredPattern {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCreateNotFoundIgnoredPatternMutationMutationVariables) => AdminCreateNotFoundIgnoredPatternMutationMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminShowNotFoundIgnoredPattern {\n\t\t\tadmin {\n\t\t\t\tallNotFoundIgnoredPatterns {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tpattern\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminShowNotFoundIgnoredPatternQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminDeleteNotFoundIgnoredPatternMutation($input: DeleteNotFoundIgnoredPatternInput!) {\n\t\t\tadmin {\n\t\t\t\tdata: deleteNotFoundIgnoredPattern(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on DeleteNotFoundIgnoredPatternPayload {\n\t\t\t\t\t\tdeletedId\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminDeleteNotFoundIgnoredPatternMutationMutationVariables) => AdminDeleteNotFoundIgnoredPatternMutationMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminUpdateNotFoundIgnoredPatternMutation($input: UpdateNotFoundIgnoredPatternInput!) {\n\t\t\tadmin {\n\t\t\t\tdata: updateNotFoundIgnoredPattern(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on UpdateNotFoundIgnoredPatternPayload {\n\t\t\t\t\t\tnotFoundIgnoredPattern {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminUpdateNotFoundIgnoredPatternMutationMutationVariables) => AdminUpdateNotFoundIgnoredPatternMutationMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminOffers {\n\t\t\tadmin {\n\t\t\t\tallOffers {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tpublicId\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tlifetime\n\t\t\t\t\t\tpriceUSD\n\t\t\t\t\t\tstartsAt\n\t\t\t\t\t\tendsAt\n\t\t\t\t\t\tsubgraphs {\n\t\t\t\t\t\t\tname\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminOffersQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminCreateOfferMutation($input: CreateOfferInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: createOffer(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on CreateOfferPayload {\n\t\t\t\t\t\toffer {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tpublicId\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCreateOfferMutationMutationVariables) => AdminCreateOfferMutationMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminShowOffer($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\toffer(id: $id) {\n\t\t\t\t\tid\n\t\t\t\t\tpublicId\n\t\t\t\t\tcreatedAt\n\t\t\t\t\tlifetime\n\t\t\t\t\tpriceUSD\n\t\t\t\t\tstartsAt\n\t\t\t\t\tendsAt\n\t\t\t\t\tsubgraphIds\n\t\t\t\t\tsubgraphs {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tname\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminShowOfferQueryVariables) => AdminShowOfferQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminUpdateOfferMutation($input: UpdateOfferInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: updateOffer(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on UpdateOfferPayload {\n\t\t\t\t\t\toffer {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tpublicId\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminUpdateOfferMutationMutationVariables) => AdminUpdateOfferMutationMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminDeletePatreonCredentials($input: DeletePatreonCredentialsInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: deletePatreonCredentials(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on ErrorPayload{\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on DeletePatreonCredentialsPayload {\n\t\t\t\t\t\tpatreonCredentials {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminDeletePatreonCredentialsMutationVariables) => AdminDeletePatreonCredentialsMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation RefreshPatreonData($input: RefreshPatreonDataInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: refreshPatreonData(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on RefreshPatreonDataPayload {\n\t\t\t\t\t\tsuccess\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: RefreshPatreonDataMutationVariables) => RefreshPatreonDataMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminRestorePatreonCredentials($input: RestorePatreonCredentialsInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: restorePatreonCredentials(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on RestorePatreonCredentialsPayload {\n\t\t\t\t\t\tpatreonCredentials {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminRestorePatreonCredentialsMutationVariables) => AdminRestorePatreonCredentialsMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminPatreonCredentials($filter: AdminPatreonCredentialsFilterInput) {\n\t\t\tadmin {\n\t\t\t\tallPatreonCredentials(filter: $filter) {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tstate\n\t\t\t\t\t\tcreatorAccessToken\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tsyncedAt\n\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminPatreonCredentialsQueryVariables) => AdminPatreonCredentialsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminCreatePatreonCreds($input: CreatePatreonCredentialsInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: createPatreonCredentials(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on CreatePatreonCredentialsPayload {\n\t\t\t\t\t\tpatreonCredentials {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCreatePatreonCredsMutationVariables) => AdminCreatePatreonCredsMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminPatreonCredentialsById($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\tpatreonCredentials(id: $id) {\n\t\t\t\t\tcreatedAt\n\t\t\t\t\tcreatorAccessToken\n\t\t\t\t\tstate\n\n\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\temail\n\t\t\t\t\t}\n\n\t\t\t\t\ttiers {\n\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tmissedAt\n\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t\tamountCents\n\n\t\t\t\t\t\t\tsubgraphs {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t\n\t\t\t\t\tmembers {\n\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\tstatus\n\t\t\t\t\t\tcurrentTier {\n\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t}\n'): (variables: AdminPatreonCredentialsByIdQueryVariables) => AdminPatreonCredentialsByIdQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminPatreoncredentialsShowSubgraphs {\n\t\t\tadmin {\n\t\t\t\tallSubgraphs {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tname\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminPatreoncredentialsShowSubgraphsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminPatreoncredentialsShowSubgraphsSave($input: SetPatreonTierSubgraphsInput!) {\n\t\t\tadmin {\n\t\t\t\tdata: setPatreonTierSubgraphs(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on SetPatreonTierSubgraphsPayload {\n\t\t\t\t\t\tsuccess\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminPatreoncredentialsShowSubgraphsSaveMutationVariables) => AdminPatreoncredentialsShowSubgraphsSaveMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminPurchases {\n\t\t\tadmin {\n\t\t\t\tallPurchases {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tpaymentProvider\n\t\t\t\t\t\tstatus\n\t\t\t\t\t\tsuccessful\n\t\t\t\t\t\tofferId\n\t\t\t\t\t\temail\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminPurchasesQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminRedirects {\n\t\t\tadmin {\n\t\t\t\tallRedirects {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tpattern\n\t\t\t\t\t\tignoreCase\n\t\t\t\t\t\tisRegex\n\t\t\t\t\t\ttarget\n\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminRedirectsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminCreateRedirectMutation($input: CreateRedirectInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: createRedirect(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on CreateRedirectPayload {\n\t\t\t\t\t\tredirect {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCreateRedirectMutationMutationVariables) => AdminCreateRedirectMutationMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminShowRedirect($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\tredirect(id: $id) {\n\t\t\t\t\tid\n\t\t\t\t\tcreatedAt\n\t\t\t\t\tpattern\n\t\t\t\t\tignoreCase\n\t\t\t\t\tisRegex\n\t\t\t\t\ttarget\n\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\tid\n\t\t\t\t\t\temail\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminShowRedirectQueryVariables) => AdminShowRedirectQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminDeleteRedirectMutation($input: DeleteRedirectInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: deleteRedirect(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on DeleteRedirectPayload {\n\t\t\t\t\t\tid\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminDeleteRedirectMutationMutationVariables) => AdminDeleteRedirectMutationMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminUpdateRedirectMutation($input: UpdateRedirectInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: updateRedirect(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on UpdateRedirectPayload {\n\t\t\t\t\t\tredirect {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminUpdateRedirectMutationMutationVariables) => AdminUpdateRedirectMutationMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminMakeReleaseLive($input: MakeReleaseLiveInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: makeReleaseLive(input:$input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t\t... on MakeReleaseLivePayload {\n\t\t\t\t\t\trelease {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminMakeReleaseLiveMutationVariables) => AdminMakeReleaseLiveMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminReleases {\n\t\t\tadmin {\n\t\t\t\tallReleases {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tcreatedBy{\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t\ttitle\n\t\t\t\t\t\tisLive\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminReleasesQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminCreateRelease($input: CreateReleaseInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: createRelease(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on CreateReleasePayload {\n\t\t\t\t\t\trelease {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCreateReleaseMutationVariables) => AdminCreateReleaseMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminListSubgraphs {\n\t\t\tadmin {\n\t\t\t\tallSubgraphs {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\t__typename\n\t\t\t\t\t\tid\n\t\t\t\t\t\tname\n\t\t\t\t\t\tcolor\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminListSubgraphsQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminSelectSubgraphList {\n\t\t\tadmin {\n\t\t\t\tallSubgraphs {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tname\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminSelectSubgraphListQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminSelectSubgraph {\n\t\t\tadmin {\n\t\t\t\tallSubgraphs {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tname\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminSelectSubgraphQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminShowSubgraph($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\tsubgraph(id: $id) {\n\t\t\t\t\tid\n\t\t\t\t\tname\n\t\t\t\t\tcolor\n\t\t\t\t\thidden\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminShowSubgraphQueryVariables) => AdminShowSubgraphQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation UpdateSubgraph($input: UpdateSubgraphInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: updateSubgraph(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on UpdateSubgraphPayload {\n\t\t\t\t\t\tsubgraph {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tcolor\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: UpdateSubgraphMutationVariables) => UpdateSubgraphMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminTelegramAccounts {\n\t\t\tadmin {\n\t\t\t\tallTelegramAccounts {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tphone\n\t\t\t\t\t\tdisplayName\n\t\t\t\t\t\tisPremium\n\t\t\t\t\t\tenabled\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminTelegramAccountsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation StartTelegramAccountAuth($input: AdminStartTelegramAccountAuthInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: startTelegramAccountAuth(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on AdminStartTelegramAccountAuthPayload {\n\t\t\t\t\t\tauthState {\n\t\t\t\t\t\t\tphone\n\t\t\t\t\t\t\tstate\n\t\t\t\t\t\t\tpasswordHint\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: StartTelegramAccountAuthMutationVariables) => StartTelegramAccountAuthMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation CompleteTelegramAccountAuth($input: AdminCompleteTelegramAccountAuthInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: completeTelegramAccountAuth(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on AdminCompleteTelegramAccountAuthPayload {\n\t\t\t\t\t\taccount {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tphone\n\t\t\t\t\t\t\tdisplayName\n\t\t\t\t\t\t\tisPremium\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: CompleteTelegramAccountAuthMutationVariables) => CompleteTelegramAccountAuthMutation

export function $trip2g_graphql_request(query: '\n\t\t\tquery AdminTelegramAccountDialogs($id: Int64!) {\n\t\t\t\tadmin {\n\t\t\t\t\ttelegramAccount(id: $id) {\n\t\t\t\t\t\tdialogs {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tusername\n\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t\ttype\n\t\t\t\t\t\t\tpublishTags {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\tpublishInstantTags {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): (variables: AdminTelegramAccountDialogsQueryVariables) => AdminTelegramAccountDialogsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminImportTelegramAccountChannel($input: AdminImportTelegramAccountChannelInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: importTelegramAccountChannel(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on AdminImportTelegramAccountChannelPayload {\n\t\t\t\t\t\tsuccess\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminImportTelegramAccountChannelMutationVariables) => AdminImportTelegramAccountChannelMutation

export function $trip2g_graphql_request(query: '\n\t\t\tquery AdminTelegramAccountShowDialogsInstantTags {\n\t\t\t\tadmin {\n\t\t\t\t\tallTelegramPublishTags {\n\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tlabel\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): () => AdminTelegramAccountShowDialogsInstantTagsQuery

export function $trip2g_graphql_request(query: '\n\t\t\tmutation AdminTelegramAccountShowDialogsInstantTagsSave($input: AdminSetTelegramAccountChatPublishInstantTagsInput!) {\n\t\t\t\tadmin {\n\t\t\t\t\tdata: setTelegramAccountChatPublishInstantTags(input: $input) {\n\t\t\t\t\t\t__typename\n\t\t\t\t\t\t... on AdminSetTelegramAccountChatPublishInstantTagsPayload {\n\t\t\t\t\t\t\tsuccess\n\t\t\t\t\t\t}\n\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): (variables: AdminTelegramAccountShowDialogsInstantTagsSaveMutationVariables) => AdminTelegramAccountShowDialogsInstantTagsSaveMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminTelegramAccountShowDialogsTags {\n\t\t\tadmin {\n\t\t\t\tallTelegramPublishTags {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tlabel\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminTelegramAccountShowDialogsTagsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminTelegramAccountShowDialogsTagsSave($input: AdminSetTelegramAccountChatPublishTagsInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: setTelegramAccountChatPublishTags(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on AdminSetTelegramAccountChatPublishTagsPayload {\n\t\t\t\t\t\tsuccess\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminTelegramAccountShowDialogsTagsSaveMutationVariables) => AdminTelegramAccountShowDialogsTagsSaveMutation

export function $trip2g_graphql_request(query: '\n\t\t\tmutation AdminSignOutTelegramAccount($input: AdminSignOutTelegramAccountInput!) {\n\t\t\t\tadmin {\n\t\t\t\t\tdata: signOutTelegramAccount(input: $input) {\n\t\t\t\t\t\t... on AdminSignOutTelegramAccountPayload {\n\t\t\t\t\t\t\tsuccess\n\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t}\n\n\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): (variables: AdminSignOutTelegramAccountMutationVariables) => AdminSignOutTelegramAccountMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminTelegramAccountUpdate($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\ttelegramAccount(id: $id) {\n\t\t\t\t\tid\n\t\t\t\t\tphone\n\t\t\t\t\tdisplayName\n\t\t\t\t\tisPremium\n\t\t\t\t\tenabled\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminTelegramAccountUpdateQueryVariables) => AdminTelegramAccountUpdateQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminUpdateTelegramAccountMutation($input: AdminUpdateTelegramAccountInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: updateTelegramAccount(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on AdminUpdateTelegramAccountPayload {\n\t\t\t\t\t\taccount {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tenabled\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminUpdateTelegramAccountMutationMutationVariables) => AdminUpdateTelegramAccountMutationMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminResetTelegramPublishNote($input: ResetTelegramPublishNoteInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: resetTelegramPublishNote(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on ResetTelegramPublishNotePayload {\n\t\t\t\t\t\tpublishNote {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminResetTelegramPublishNoteMutationVariables) => AdminResetTelegramPublishNoteMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminSendTelegramPublishNoteNow($input: SendTelegramPublishNoteNowInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: sendTelegramPublishNoteNow(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on SendTelegramPublishNoteNowPayload {\n\t\t\t\t\t\tpublishNote {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminSendTelegramPublishNoteNowMutationVariables) => AdminSendTelegramPublishNoteNowMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminTelegramPublishNoteCount($filter: AdminTelegramPublishNotesFilter!) {\n\t\t\tadmin {\n\t\t\t\tallTelegramPublishNotes(filter: $filter) {\n\t\t\t\t\tcount\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminTelegramPublishNoteCountQueryVariables) => AdminTelegramPublishNoteCountQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminTelegramPublishNotes($filter: AdminTelegramPublishNotesFilter!) {\n\t\t\tadmin {\n\t\t\t\tallTelegramPublishNotes(filter: $filter) {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tpublishAt\n\t\t\t\t\t\tsecondsUntilPublish\n\t\t\t\t\t\tpublishedAt\n\t\t\t\t\t\tstatus\n\t\t\t\t\t\terrorCount\n\t\t\t\t\t\tnoteView {\n\t\t\t\t\t\t\ttitle\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminTelegramPublishNotesQueryVariables) => AdminTelegramPublishNotesQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminTelegramPublishNote($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\ttelegramPublishNote(id: $id) {\n\t\t\t\t\tid\n\t\t\t\t\tcreatedAt\n\t\t\t\t\tpublishAt\n\t\t\t\t\tsecondsUntilPublish\n\t\t\t\t\tpublishedAt\n\t\t\t\t\tstatus\n\t\t\t\t\ttags {\n\t\t\t\t\t\tlabel\n\t\t\t\t\t}\n\t\t\t\t\tchats {\n\t\t\t\t\t\tchatTitle\n\t\t\t\t\t\tchatType\n\t\t\t\t\t}\n\t\t\t\t\tnoteView {\n\t\t\t\t\t\ttitle\n\t\t\t\t\t}\n\t\t\t\t\tpost {\n\t\t\t\t\t\tcontent\n\t\t\t\t\t\twarnings\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminTelegramPublishNoteQueryVariables) => AdminTelegramPublishNoteQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminTgBots {\n\t\t\tadmin {\n\t\t\t\tallTgBots {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tname\n\t\t\t\t\t\tdescription\n\t\t\t\t\t\tenabled\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminTgBotsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminCreateTgBotMutation($input: CreateTgBotInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: createTgBot(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on CreateTgBotPayload {\n\t\t\t\t\t\ttgBot {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tname\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCreateTgBotMutationMutationVariables) => AdminCreateTgBotMutationMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminTgBotChats($filter: AdminTgBotChatsFilterInput!) {\n\t\t\tadmin {\n\t\t\t\ttgBotChats(filter: $filter) {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tchatType\n\t\t\t\t\t\tchatTitle\n\t\t\t\t\t\taddedAt\n\t\t\t\t\t\tremovedAt\n\t\t\t\t\t\tmemberCount\n\t\t\t\t\t\tsubgraphAccesses {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tsubgraphId\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminTgBotChatsQueryVariables) => AdminTgBotChatsQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminTgbotShowChatsSubgraphs {\n\t\t\tadmin {\n\t\t\t\tallSubgraphs {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tname\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminTgbotShowChatsSubgraphsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminTgbotsShowchatsSubgraphsSave($input: SetTgChatSubgraphsInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: setTgChatSubgraphs(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on SetTgChatSubgraphsPayload {\n\t\t\t\t\t\tsuccess\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminTgbotsShowchatsSubgraphsSaveMutationVariables) => AdminTgbotsShowchatsSubgraphsSaveMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminTgBotInviteChats($filter: AdminTgBotChatsFilterInput!) {\n\t\t\tadmin {\n\t\t\t\ttgBotChats(filter: $filter) {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tchatType\n\t\t\t\t\t\tchatTitle\n\t\t\t\t\t\taddedAt\n\t\t\t\t\t\tremovedAt\n\t\t\t\t\t\tmemberCount\n\t\t\t\t\t\tsubgraphInvites {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tsubgraphId\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminTgBotInviteChatsQueryVariables) => AdminTgBotInviteChatsQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminTgbotShowInviteChatsSubgraphs {\n\t\t\tadmin {\n\t\t\t\tallSubgraphs {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tname\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminTgbotShowInviteChatsSubgraphsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminTgbotShowInviteChatsSubgraphsSave($input: SetTgChatSubgraphInvitesInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: setTgChatSubgraphInvites(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on SetTgChatSubgraphInvitesPayload {\n\t\t\t\t\t\tsuccess\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminTgbotShowInviteChatsSubgraphsSaveMutationVariables) => AdminTgbotShowInviteChatsSubgraphsSaveMutation

export function $trip2g_graphql_request(query: '\n\t\t\tquery AdminTgbotShowPublishInstantTags {\n\t\t\t\tadmin {\n\t\t\t\t\tallTelegramPublishTags {\n\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tlabel\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): () => AdminTgbotShowPublishInstantTagsQuery

export function $trip2g_graphql_request(query: '\n\t\t\tmutation AdminTgbotShowPublishInstantTagsSave($input: SetTgChatPublishInstantTagsInput!) {\n\t\t\t\tadmin {\n\t\t\t\t\tdata: setTgChatPublishInstantTags(input: $input) {\n\t\t\t\t\t\t__typename\n\t\t\t\t\t\t... on SetTgChatPublishInstantTagsPayload {\n\t\t\t\t\t\t\tsuccess\n\t\t\t\t\t\t}\n\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): (variables: AdminTgbotShowPublishInstantTagsSaveMutationVariables) => AdminTgbotShowPublishInstantTagsSaveMutation

export function $trip2g_graphql_request(query: '\n\t\t\tquery AdminTgBotPublishTags($filter: AdminTgBotChatsFilterInput!) {\n\t\t\t\tadmin {\n\t\t\t\t\ttgBotChats(filter: $filter) {\n\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tchatType\n\t\t\t\t\t\t\tchatTitle\n\t\t\t\t\t\t\taddedAt\n\t\t\t\t\t\t\tremovedAt\n\t\t\t\t\t\t\tmemberCount\n\t\t\t\t\t\t\tpublishTags {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\tpublishInstantTags {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): (variables: AdminTgBotPublishTagsQueryVariables) => AdminTgBotPublishTagsQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminTgbotShowPublishTags {\n\t\t\tadmin {\n\t\t\t\tallTelegramPublishTags {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tlabel\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminTgbotShowPublishTagsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminTgbotShowPublishTagsSave($input: SetTgChatPublishTagsInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: setTgChatPublishTags(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on SetTgChatPublishTagsPayload {\n\t\t\t\t\t\tsuccess\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminTgbotShowPublishTagsSaveMutationVariables) => AdminTgbotShowPublishTagsSaveMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminShowTgBot($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\ttgBot(id: $id) {\n\t\t\t\t\tid\n\t\t\t\t\tname\n\t\t\t\t\tdescription\n\t\t\t\t\tenabled\n\t\t\t\t\tcreatedAt\n\t\t\t\t\tcreatedBy {\n\t\t\t\t\t\temail\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminShowTgBotQueryVariables) => AdminShowTgBotQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminUpdateTgBotMutation($input: UpdateTgBotInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: updateTgBot(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on UpdateTgBotPayload {\n\t\t\t\t\t\ttgBot {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tdescription\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminUpdateTgBotMutationMutationVariables) => AdminUpdateTgBotMutationMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminListUserBans {\n\t\t\tadmin {\n\t\t\t\tallUserUserBans {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid: userId\n\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t\tbannedBy {\n\t\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\treason\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminListUserBansQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminBanUser($input: BanUserInput!) {\n\t\t\tadmin {\n\t\t\t\tbanUser(input: $input) {\n\t\t\t\t\t... on BanUserPayload {\n\t\t\t\t\t\t__typename\n\t\t\t\t\t\tuser { id, __typename }\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t__typename\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminBanUserMutationVariables) => AdminBanUserMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminUnbanUser($input: UnbanUserInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: unbanUser(input: $input) {\n\t\t\t\t\t... on UnbanUserPayload {\n\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminUnbanUserMutationVariables) => AdminUnbanUserMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminListUsers {\n\t\t\tadmin {\n\t\t\t\tallUsers {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\temail\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tban { reason }\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminListUsersQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminCreateUser($input: CreateUserInput!) {\n\t\t\tadmin {\n\t\t\t\tcreateUser(input: $input) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on CreateUserPayload {\n\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminCreateUserMutationVariables) => AdminCreateUserMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminUserShow($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\tuser(id: $id) {\n\t\t\t\t\tid\n\t\t\t\t\temail\n\t\t\t\t\tcreatedAt\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminUserShowQueryVariables) => AdminUserShowQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminUserSubgraphAccess($id: Int64!) {\n\t\t\tadmin {\n\t\t\t\tallSubgraphs {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tname\n\t\t\t\t\t}\n\t\t\t\t}\n\n\t\t\t\tuserSubgraphAccess(id: $id) {\n\t\t\t\t\tuserId\n\t\t\t\t\tsubgraphId\n\t\t\t\t\texpiresAt\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminUserSubgraphAccessQueryVariables) => AdminUserSubgraphAccessQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation AdminUpdateUserSubgraphAccess($input: UpdateUserSubgraphAccessInput!) {\n\t\t\tadmin {\n\t\t\t\tpayload: updateUserSubgraphAccess(input: $input) {\n\t\t\t\t\t... on UpdateUserSubgraphAccessPayload {\n\t\t\t\t\t\tuserSubgraphAccess {\n\t\t\t\t\t\t\t__typename\n\t\t\t\t\t\t\texpiresAt\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\tmessage\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: AdminUpdateUserSubgraphAccessMutationVariables) => AdminUpdateUserSubgraphAccessMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminListUserSubgraphAccesses {\n\t\t\tadmin {\n\t\t\t\tdata: allUserSubgraphAccesses {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\t__typename\n\t\t\t\t\t\tid\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\texpiresAt\n\t\t\t\t\t\tsubgraph {\n\t\t\t\t\t\t\tname\n\t\t\t\t\t\t}\n\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\temail\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminListUserSubgraphAccessesQuery

export function $trip2g_graphql_request(query: '\n\t\t\tquery AdminUserEditQuery($id: Int64!) {\n\t\t\t\tadmin {\n\t\t\t\t\tuser(id: $id) {\n\t\t\t\t\t\tid\n\t\t\t\t\t\temail\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t'): (variables: AdminUserEditQueryQueryVariables) => AdminUserEditQueryQuery

export function $trip2g_graphql_request(query: '\n\t\t\tmutation AdminUpdateUser($input: UpdateUserInput!) {\n\t\t\t\tadmin {\n\t\t\t\t\tupdateUser(input: $input) {\n\t\t\t\t\t\t__typename\n\t\t\t\t\t\t... on UpdateUserPayload {\n\t\t\t\t\t\t\tuser {\n\t\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\t\temail\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t\t\tmessage\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t}\n\t\t}\n\t\t'): (variables: AdminUpdateUserMutationVariables) => AdminUpdateUserMutation

export function $trip2g_graphql_request(query: '\n\t\tquery AdminWaitListEmailRequests {\n\t\t\tadmin {\n\t\t\t\tallWaitListEmailRequests {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\temail\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tip\n\t\t\t\t\t\tnotePath\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminWaitListEmailRequestsQuery

export function $trip2g_graphql_request(query: '\n\t\tquery AdminWaitListTgBotRequests {\n\t\t\tadmin {\n\t\t\t\tallWaitListTgBotRequests {\n\t\t\t\t\tnodes {\n\t\t\t\t\t\tchatId\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\tnotePathId\n\t\t\t\t\t\tnotePath\n\t\t\t\t\t\tbotName\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => AdminWaitListTgBotRequestsQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation SignOut {\n\t\t\tdata: signOut {\n\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t__typename\n\t\t\t\t\tmessage\n\t\t\t\t}\n\t\t\t\t... on SignOutPayload {\n\t\t\t\t\t__typename\n\t\t\t\t\tviewer {\n\t\t\t\t\t\tid\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => SignOutMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation RequestEmailSignInCode($input: RequestEmailSignInCodeInput!) {\n\t\t\tdata: requestEmailSignInCode(input: $input) {\n\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t__typename\n\t\t\t\t\tmessage\n\t\t\t\t}\n\t\t\t\t... on RequestEmailSignInCodePayload {\n\t\t\t\t\t__typename\n\t\t\t\t\tsuccess\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: RequestEmailSignInCodeMutationVariables) => RequestEmailSignInCodeMutation

export function $trip2g_graphql_request(query: '\n\t\tmutation SignInByEmail($input: SignInByEmailInput!) {\n\t\t\tdata: signInByEmail(input: $input) {\n\t\t\t\t... on SignInPayload {\n\t\t\t\t\t__typename\n\t\t\t\t\ttoken\n\t\t\t\t}\n\t\t\t\t... on ErrorPayload {\n\t\t\t\t\t__typename\n\t\t\t\t\tmessage\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: SignInByEmailMutationVariables) => SignInByEmailMutation

export function $trip2g_graphql_request(query: '\n\t\tquery Viewer {\n\t\t\tviewer {\n\t\t\t\tid\n\t\t\t\trole\n\t\t\t\tuser {\n\t\t\t\t\temail\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => ViewerQuery

export function $trip2g_graphql_request(query: '\n\t\tquery ReaderQuery($input: NoteInput!) {\n\t\t\tnote(input: $input) {\n\t\t\t\ttitle\n\t\t\t\thtml\n\t\t\t\tpathId\n\t\t\t\ttoc {\n\t\t\t\t\tid\n\t\t\t\t\ttitle\n\t\t\t\t\tlevel\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: ReaderQueryQueryVariables) => ReaderQueryQuery

export function $trip2g_graphql_request(query: '\n\t\tquery FavoriteNotes {\n\t\t\tviewer {\n\t\t\t\tuser {\n\t\t\t\t\tfavoriteNotes {\n\t\t\t\t\t\tpathId\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => FavoriteNotesQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation ToggleFavoriteNote($input: ToggleFavoriteNoteInput!) {\n\t\t\tpayload: toggleFavoriteNote(input: $input) {\n\t\t\t\t__typename\n\t\t\t\t... on ToggleFavoriteNotePayload {\n\t\t\t\t\tfavoriteNotes {\n\t\t\t\t\t\tpathId\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t\t... on ErrorPayload {\n\t\t\t\t\tmessage\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: ToggleFavoriteNoteMutationVariables) => ToggleFavoriteNoteMutation

export function $trip2g_graphql_request(query: '\n\t\tquery PaywallActivePurchaseQuery {\n\t\t\tviewer {\n\t\t\t\tactivePurchases {\n\t\t\t\t\tid\n\t\t\t\t\tstatus\n\t\t\t\t\tsuccessful\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => PaywallActivePurchaseQueryQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation CreateEmailWaitListRequestMutation ($input: CreateEmailWaitListRequestInput!) {\n\t\t\tcreateEmailWaitListRequest(input: $input) {\n\t\t\t\t__typename\n\t\t\t\t... on ErrorPayload {\n\t\t\t\t\tmessage\n\t\t\t\t}\n\t\t\t\t... on CreateEmailWaitListRequestPayload {\n\t\t\t\t\tsuccess\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: CreateEmailWaitListRequestMutationMutationVariables) => CreateEmailWaitListRequestMutationMutation

export function $trip2g_graphql_request(query: '\n\t\tquery PaywallQuery($filter: ViewerOffersFilter!) {\n\t\t\tviewer {\n\t\t\t\toffers(filter: $filter) {\n\t\t\t\t\t__typename\n\t\t\t\t\t... on ActiveOffers {\n\t\t\t\t\t\tnodes {\n\t\t\t\t\t\t\tid\n\t\t\t\t\t\t\tpriceUSD\n\t\t\t\t\t\t\tsubgraphs {\n\t\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t\t... on SubgraphWaitList {\n\t\t\t\t\t\ttgBotUrl\n\t\t\t\t\t\temailAllowed\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: PaywallQueryQueryVariables) => PaywallQueryQuery

export function $trip2g_graphql_request(query: '\n\t\tmutation CreatePaymentLink($input: CreatePaymentLinkInput!) {\n\t\t\tdata: createPaymentLink(input: $input) {\n\t\t\t\t__typename\n\t\t\t\t... on CreatePaymentLinkPayload {\n\t\t\t\t\tredirectUrl\n\t\t\t\t}\n\t\t\t\t... on ErrorPayload {\n\t\t\t\t\tmessage\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: CreatePaymentLinkMutationVariables) => CreatePaymentLinkMutation

export function $trip2g_graphql_request(query: '\n\t\tquery SiteSearch($input: SearchInput!) {\n\t\t\tsearch(input: $input) {\n\t\t\t\tnodes {\n\t\t\t\t\thighlightedTitle\n\t\t\t\t\thighlightedContent\n\t\t\t\t\tid: url\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): (variables: SiteSearchQueryVariables) => SiteSearchQuery

export function $trip2g_graphql_request(query: '\n\t\tquery UserSubscriptions {\n\t\t\tviewer {\n\t\t\t\tuser {\n\t\t\t\t\tsubgraphAccesses {\n\t\t\t\t\t\tid\n\t\t\t\t\t\tcreatedAt\n\t\t\t\t\t\texpiresAt\n\t\t\t\t\t\tsubgraph {\n\t\t\t\t\t\t\tname\n\t\t\t\t\t\t\thomePath\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t'): () => UserSubscriptionsQuery

export function $trip2g_graphql_request(query: any) { return $trip2g_graphql_raw_request(query); }

export function $trip2g_graphql_subscription(query: any, variables?: any) { return $trip2g_graphql_raw_subscription(query, variables); }



export const $trip2g_graphql_persist_queries = {"Admins":"31b7b9c10ee6d0342eb987451d91aa3385bfe05d179bc9d6f78cb5c170b3109d","DisableApiKey":"e834a198cb4b6cc4e061cfd9a4d9aa98eb2c56ad1552a51cfa01d1fa3294e00d","AdminListApiKeys":"d59e369cd035f94198e7e9f2bc106e8db3c4fff2230e092b3763060130ad7c0d","AdminCreateApiKey":"a2170d5503be730c920e1749a1277261d594cbd3f2338b3c63ac41bcb63fc7d8","AdminApiKeyShowQuery":"7b2ca3f35170292de2930efaa89e1ef5c6fedd2f69612e2819657e72584d79a9","AdminAuditLogs":"1a082b80a9018c133e5b8dbb36ff99f1eea34ff8fdc8515e9a56042aa2aedc60","AdminClearBackgroundQueue":"db89b56d9675982a013fd0a4d7430aea6f090194c04fe0b7bcbd8fccbd8f7ee0","AdminStartBackgroundQueue":"2f81310580c2f0fb7bdff12f31e1a8919f0a2dd536245ea4e183be40a9f8747b","AdminStopBackgroundQueue":"82b19096c8dcac82dfdeb84afbbb446280269846daae5d26761b02a482be8b79","AdminBackgroundQueues":"c38f2e764f1302b28748a2cbccdf44bffd62403b0ef524e598decc6f5c471fca","AdminBackgroundQueue":"e7773b5dfc8c0760bdb2b96d6facad42f397081a0c21de1e8e693a43172c606d","AdminDeleteBoostyCredentials":"3d0823a1b5aeea640b9da428da98caea95a7e8e3d6b18c2ec3c9bc0c00a7ffd5","RefreshBoostyData":"61acb201718f9780deaacd13cc05397dfa1e05a27bd1ada5fe1ae45ffacbda30","AdminRestoreBoostyCredentials":"090d1e2ed8132bb6cf8d362449514c9e46a88b8c9c3b9eb931ba57fece576dc4","AdminBoostyCredentials":"25c080897f9450a5a8fac63762e30f821774e6ce857a3c2351428fb6d6f186a2","AdminCreateBoostyCreds":"4e5b4046aa6ced6ef375f9c9d50c17ee276a4bef3b9e8add04cbfd68a4095a0f","AdminBoostyCredentialsById":"600723d3709db0ec905cecbc765d16c70ea0e39599872800c79618c9d3ea85d9","AdminBoostycredentialsShowSubgraphs":"0f4837d0dbe327b61f7968350094b7c6728dcf8a328188169f1ddc1df26d3e70","AdminBoostycredentialsShowSubgraphsSave":"afc4036233eb030a18f9008ecb7d84c458daef51d9a8bf2c5f7f6396c0370d04","AdminConfigVersions":"11084301e54ab0af56694fa9e7a9349b8c959b45fdea6f2c827154065ec2f0e1","AdminCreateConfigLatestConfig":"95fab47491a0e74762bad6eae78778a4f43363360be191a73cd493edd5655246","AdminCreateConfigVersion":"599e86e012056322485099caff4c74b7ce165ec6aff05e9edeff0899aaa8014c","AdminRunCronJob":"73fe25ff98dbcdaa87a886f7d13c75f7ffe76a9048206bfdd6f08ac526a21ca8","AdminAllCronJobs":"f5f63f4f2278f386473d9c48f85b5f4c892afa6b6f1f6d686172ec9ec42bbc82","AdminCronJobExecutions":"b7948c64a8dcefbbdb118e876b939034d89e930e92728b76280fe802709e065c","AdminCronJobShow":"1fd178aee76ce85c0b4fd89b1c2483decce36187def4e8b728ec298378f3df14","AdminCronJobUpdate":"e07bfb2906838f35ba3b84db68aaecc9fce8d039bd2661a0963432c5203e7540","AdminUpdateCronJob":"0460e7f0fe223c77a594e150679fc7dac589ae3683b6561e4dcf201c1b24b6fe","AdminBuildInfo":"58951e6413181734d969d6ae3a9b2c37e7e37d0fd7b53a41ef63aed2cde8daac","AdminRecentlyModifiedNotes":"c990aab7efd50d9c5f08aa834b361cbf185e60a156e4f985bf4f144db0973751","DisableGitToken":"9c25101951f62812b1b5f43845e7925e6c779d85b9db019a29d67171a77b3606","AdminGitTokens":"6def4f2bfa2ae2af52361495dc476322d1bdf3f668755576ac98c69e42d44f80","AdminCreateGitToken":"c77c2ffe6a0ed90fa7d3e6546ad9fab5d516e99f8f52b560c46e6233b9251855","AdminHealthChecks":"6964f2657595c376d2ed7613376870a4c9df5f07f84441f9bb6ed440f1400dc7","AdminDeleteHtmlInjection":"ae499ffc5c4e3a0effac670cd84adda14cd1573333d2e193ed90901aa56bd438","AdminHtmlInjections":"79ba1ce7ee04726dbb1b7a64ba86d6d56f42f98defa44ec8ea77d48996972a44","AdminCreateHtmlInjectionMutation":"0d3cd80d4fb88be22d74efc570797fdfa7889e77f49f09a50c8bf61c07c15338","AdminShowHtmlInjection":"d1727312046a0f7ec9fa7c976c3ee5b6fe7094ba765c29dde63c6d523e274b94","AdminUpdateDataHtmlInjection":"8f9cadd42bf1df9f09c137e3e03937be4696b409c18b4e152d6702ee5cec5f0f","AdminUpdateHtmlInjection":"6499c958bc268e2a2b0684437e009f50a025f80b015473f81b317ab9af03e267","AdminNoteAssets":"71a80b7702d3fcb06d723818802e51a8f4683408e31b03650412844748685f88","AdminNoteAsset":"4125392777e418cddd74406af3ef384a49eba5794ca39a79e92787e8d2df71b3","AdminListNoteViews":"0750d17bf811e2f7bc48ae7b3857e345519ba374a98adb18a5a982644b71c1f7","AdminGraph":"b843c79d94c52d96087513a2768e3a2f1732c9a86d6ef27d97be9e93750bd332","UpdateNoteGraphPositions":"86e2eb3c75b9c50e011a46b6e3723cd97ebfe5e88f7b3c47e1e8e23566ea0a63","AdminSelectNoteView":"9739a047b122716296664c2d10ad92e31b8b3c6367ae1927a54b2468ba349b25","AdminNoteView":"0b01d78ceadccc26319d47b4d4c85400cd2be93bc5cbd0ebc84482780fae73a8","AdminNoteWarnings":"1761520561721485891ab1b2cdd5e54a0d27ced539ce95c5c70e877dafb9e23e","AdminResetNotFoundPath":"3be560968348f9adb9e5797c65b11caa913adf5db7ea3e47113591ec441796b9","AdminNotFoundPaths":"4d3dde4e044f707f70109b48bf5b12853e8a407ec5b439d8011da5ec5113be92","AdminShowNotFoundPath":"148013382b0a14cf65448fd68f05cec947bb19d9e337cc931e6f89aa4735e6ac","AdminDeleteNotFoundIgnoredPattern":"e5a6db3369038ca64e498d1f49f297c005bc719eb5e8c1ae4be5f46d9ea5523a","AdminNotFoundIgnoredPatterns":"d4146534b56631bd5736e90599f12702e1b7419199e03706244afd07789db0e2","AdminCreateNotFoundIgnoredPatternMutation":"fed2411c47a1e72a6913857c3db536e66e84323daf400c53787a0eaa3309ddcc","AdminShowNotFoundIgnoredPattern":"e2cb9295a351038a0984f23a599dfe4da435d518a0b0fe866e21f7556b37c94c","AdminDeleteNotFoundIgnoredPatternMutation":"95e5d380983cc3c2e1941b630f89e46c770b00acb4efa06520c1d7e8da5ccf68","AdminUpdateNotFoundIgnoredPatternMutation":"42c55aca562a687cef0bd7055b7ff7a856441e5638263d394761fa46f2bfad80","AdminOffers":"67c8a75420630b288603cd670ee7b7b0d1f4dae99a6bfc850490fa70cc402592","AdminCreateOfferMutation":"9a7b80fc55ee31e3ed9bc89cca38c430d4ccc638f3178c3915901f5c06c606b9","AdminShowOffer":"b4b0730aaea13ee9837ac89cae8fae39c5d4aedfde90511e144d3c99f122b8be","AdminUpdateOfferMutation":"fa85da58878c033a66304433b230ae409242e86d8e4cd5ec5961581696efaf71","AdminDeletePatreonCredentials":"d9b0d57409c0f708110ca4ffc5b16b894e57296bbadebdd9f095ff836cad717a","RefreshPatreonData":"a8e2c68cfc11cb6dee15d381ad64b90f8affafaecd80ae4cf5243dfdf79d86bd","AdminRestorePatreonCredentials":"0e8bfe8d4965f71bd48d1a2caeba1375cdecb2cf5743635e848b31b88267e7ea","AdminPatreonCredentials":"f52852dd95a63d271b3760b72e00c5a11cebf70f07f2a4c7dabcacb833186942","AdminCreatePatreonCreds":"a93585e99935493cb94147a8bd919cb1fcbe6eac2164ff18a056341ea3c91e1f","AdminPatreonCredentialsById":"6ab08480f36bdc278d86a755a994eb18053bea5ac61f1da0da15203a79872e26","AdminPatreoncredentialsShowSubgraphs":"2d9c3b59a29a6cd02b40a246d90491d6e81d8ecf44c4bf8333610d0ff0894a2b","AdminPatreoncredentialsShowSubgraphsSave":"c622724ecc6dd0d147cae6049d2bc052af809fa672deece0add30b3a2609b81f","AdminPurchases":"1793e7b78b882135bb3539e185cf1a05580cd039398c40e3c821ce31b668ef7c","AdminRedirects":"065f6c26aae03086af98536d0dcffd0af1e6c26ec78d6bd0bfb7d804b77abab5","AdminCreateRedirectMutation":"ab0062cdcf64364e2729d50b4d3edc2eb0f28de74a81413c9268116fc18ff821","AdminShowRedirect":"0b9899a7d1c06acec4a43512e6099fa239dbf133f2c92ede155fc029bbe03189","AdminDeleteRedirectMutation":"028aa4c93df10d59a0199b134dc6cc1d969e79f5b36f7a991502417df37bea63","AdminUpdateRedirectMutation":"58ac7bc9b0653a2eb0ff1df7317eefb92eea73f60dc5af7e64cdfaadf7c19731","AdminMakeReleaseLive":"f09d1bf08994771a5812fbab10cc35c074136c56f3a947fd446d1f2a393bc1d9","AdminReleases":"8a72ee8f42e12a7a72260cf6fe9b6bbaee7d695296a21e878cf1196a80000111","AdminCreateRelease":"5853ac63ee8c6f0c73e0bbf96c61c1463c9e8db0a4aaf77624a4bc92e0064d03","AdminListSubgraphs":"7e21e2c649f89e31277158ceab6af158e25ce8ca0c66dab331485ed72e181dfc","AdminSelectSubgraphList":"867c73ab0c102a05db2d31123506cb6973caf9eb19849d7e613068a685187549","AdminSelectSubgraph":"52faf3aa57ac3311a4cf93b5ac60f728f67a1a0601220c77732e632640f38f10","AdminShowSubgraph":"92032e177122bb3ae0bc2ce470ab9f2c03d9a558339708f7e34cd07b7a739d08","UpdateSubgraph":"99c35b5884cff26f3227d22441a021d53924b38555e3ebb3dce6877ceab48a30","AdminTelegramAccounts":"823027a6413d9457b2ea78fdc17e96f3d3d7e65b89c45ba147e9bdee66e614cb","StartTelegramAccountAuth":"f44006ab412fadbcd4f37eafcd67bd4635a0b9e1fdb0a2a420e06479068ce832","CompleteTelegramAccountAuth":"3db8d63d093586f036683377aaf9bb95545bf1594fada4104fb1c2008b702a20","AdminTelegramAccountDialogs":"1e0c01a7960868635ee21d107169ac475809ff0f75c1ee88c47ba957e0d8c332","AdminImportTelegramAccountChannel":"aa3d7e56c47f9a8f4b2f3209098106bc3f380ac05439d5a274feaa3d5ed46fdb","AdminTelegramAccountShowDialogsInstantTags":"d9cb29f94cda3f0070d0c6ee92d0ad2944482e89a43373203cf78389ee220c6a","AdminTelegramAccountShowDialogsInstantTagsSave":"3198e5415c24f8e37f0c18669d473c4b5ac956c91b2b52240773ebe56808f8e2","AdminTelegramAccountShowDialogsTags":"c13ef5bf0a56e1626e6dc1da45de3a2a29e23461db35c8b4d77fd6d1bb580c99","AdminTelegramAccountShowDialogsTagsSave":"be25bf420a207a797642454d8f173c0a2799d61c7f3684792a443a25801d2191","AdminSignOutTelegramAccount":"d126faf062d914f8d772ec0929b17c598f822f103e973b1e3fc8354664847cad","AdminTelegramAccountUpdate":"09dabbb216687efbca24568b8814d0405d912891d77ccd9b00f15aaec73c3896","AdminUpdateTelegramAccountMutation":"85f6aa4153aad281a8477f087224c6c61f7ee50bfd7ea9454c08e3391ec18e74","AdminResetTelegramPublishNote":"8b8c0111dc74e29ed18f3b68dcf721756e2d52ad653a11d6e019b3d421f7bc4e","AdminSendTelegramPublishNoteNow":"5b547629781b70847c504811a74f10d8c2cdf488c872033cfb3cb116f6dd7dcd","AdminTelegramPublishNoteCount":"7b72b8fcfc9bde3d6ea209c58f6266a5b027271d8d7c9a7499ef64492cf98cf3","AdminTelegramPublishNotes":"1a2dcf5b167fdada3d83ca707f9afeac488f2f4a995e8b7b4a31061be891bd73","AdminTelegramPublishNote":"c5182bb1cee9143fc6e1fbd5cb66c44a63d06751554b28ef3f3f5a519473a95b","AdminTgBots":"d7fabe798197e90df76f04f6afe8af9eef1b9da6cae7a6059afa0126c8aeaa92","AdminCreateTgBotMutation":"f8fb2ca52d215f00fbf111f8860b042f6d2adfafe40acd13a2a564cd2904726e","AdminTgBotChats":"710f5f761a43fc1edf9f7754e4b0111ce547fe049a6f078544f3705babe89234","AdminTgbotShowChatsSubgraphs":"a8aab396ad86724e5333e78ad9e71bb3f170700f9e70e6c13760e3dd60d4c7df","AdminTgbotsShowchatsSubgraphsSave":"55a009736a1249932c88ba54268a58ae13b868d0679159bae0c62390c2fb5fca","AdminTgBotInviteChats":"da6a44f35e1ca767093837eebcfe4846bb4674d6f03665302bd444360e098224","AdminTgbotShowInviteChatsSubgraphs":"95eddf5cabffdb9900906b2c4637031b7f2de1441a6c90cf23c75d988aa47d25","AdminTgbotShowInviteChatsSubgraphsSave":"1e871ffcfb0c44803bb02ce90adb3c5444c459c38cf30ef23b4ef04766cbf12c","AdminTgbotShowPublishInstantTags":"0b41ad16a2f936719eee6f209bbe243727aeed842961ef408507307533bfff22","AdminTgbotShowPublishInstantTagsSave":"55845592ad7292229fcd8e90f80b95ae8128dc2394ba2b6f07352912f63932de","AdminTgBotPublishTags":"c75b0f890e59788a0d0076a2e67fe2b74430c201074b13f751a77590a6767012","AdminTgbotShowPublishTags":"9970ddc0b0f15902549a9e824e581c69c9939c1cf241825da97563000ab09ee6","AdminTgbotShowPublishTagsSave":"75e6b0280009d01328280e3a4fd9b93844695308424eaabe6d7eeb923ce999bd","AdminShowTgBot":"6d99ca3f745b6b02a4d8e21e72620eef5d84d27effcbd63ae3055e40797418b3","AdminUpdateTgBotMutation":"0324b130d88743b356749ef9b7d9aaaf2461e08dcf9511d10550c648ff237eb7","AdminListUserBans":"6040d70a10b3382a3831068918a788a213dee3374a9efbf1f74dcf252a7b45e8","AdminBanUser":"c4a45b44b3642384d4e28936d7ddfa82f5b2b4504f873c694be3f0369c545512","AdminUnbanUser":"6d2f4cf8341515d7b6d96df9815f4c72b044e05b77e29ede9e45873ab8efd602","AdminListUsers":"94a761f818704004a943609dd89aa84af34036425d597b779b91b574e13aa46d","AdminCreateUser":"9e8192928cf6ac787508c63ea8ce9dcf820787ecb1dcae0444d5bd4b5d27f94f","AdminUserShow":"cd00ba703304fd1743cce6b5fc9508191b43a2c68298b52c6d7fa98e93127bd5","AdminUserSubgraphAccess":"cf4b04345f46aa7118228621cb393b6ecf0720159301308ce29a41cfdefb6da0","AdminUpdateUserSubgraphAccess":"fdcb57d0a19494fd015fe16d95c1657e50fa130ff19e2e2dde89a5ab4d1025b7","AdminListUserSubgraphAccesses":"dcd373f4fc744041cf1d3cd6d96aadc49d2f633305ecde0973283fe5dde1e3b2","AdminUserEditQuery":"e5a4a4da29de00348707415a78c5982d1c7bad95bf43aecff9fe935f9de553ef","AdminUpdateUser":"f79fd3f160176c393f341411c19068223f4cd4d3083382add6eb3eeb654a8b60","AdminWaitListEmailRequests":"f7df2fc974a6048b1e8d9e7b839963b14a32e4ecd46d769bb37f3a843be4fc7e","AdminWaitListTgBotRequests":"67ba998693c036677fb6933b96e6a39c905f209e433c653448832e127b8d8aad","SignOut":"ce93de9e93d241aa23f7f1fc875c14a9529ae881014cca4da3844f1f0c9b019d","RequestEmailSignInCode":"092d8a0ea5dd092a91fdfff67ea796bb03a6a86c7951832e55d9dbe034290add","SignInByEmail":"9fd9d325614b814c0c8925114e01bb9e14b32530d359e12ec5dc073f4519358c","Viewer":"653f9a99dddb2deee894084f53713e9452caea239eb7e087656dbc2aae7c555d","ReaderQuery":"01e92181e52161f23624ca6dc445ac0316c0a8e53253c3e21cf91fd60aa71a7c","FavoriteNotes":"95149858dee6f011bbad69b07f10b51110a8ac1f24f849547f7feeed617b4dce","ToggleFavoriteNote":"de3e0de0e3d014e1e9e11daf108e93f204f45dbe7bd90956979e124d1ea76790","PaywallActivePurchaseQuery":"4884e87c0c2a528c8fc7e42c8914c0327d114cbd7be6bf4e719a7c3fbb9fe5fb","CreateEmailWaitListRequestMutation":"f9af5df41fe86e9363f570765ab1833f3ce56a47f1d3fbef54502dfa0f74707a","PaywallQuery":"2a77280606a1905203b68482e752e3b6bad4c0ddf76b7b01cd7f60c9218692fb","CreatePaymentLink":"ac8b21f2dc1e8f8ecdb847d0102b2b6a83dba671692e790f0ab0baf14a4facf4","SiteSearch":"a826bb24b493c8a5e3bc3919c5e9595dd4e2c36d4e0cbf3db3cf30ee2e9819f0","UserSubscriptions":"f703829d1cbd3c9ece3c9f248da8b91ff1bde0dbdff42f0b8ae5faee195a26f0"}

export const $trip2g_graphql_admin_telegram_account_auth_state_enum = AdminTelegramAccountAuthStateEnum;

export const $trip2g_graphql_admin_telegram_account_dialog_type = AdminTelegramAccountDialogType;

export const $trip2g_graphql_audit_log_level_enum = AuditLogLevelEnum;

export const $trip2g_graphql_boosty_credentials_state_enum = BoostyCredentialsStateEnum;

export const $trip2g_graphql_cron_job_execution_status = CronJobExecutionStatus;

export const $trip2g_graphql_health_check_status = HealthCheckStatus;

export const $trip2g_graphql_note_warning_level_enum = NoteWarningLevelEnum;

export const $trip2g_graphql_patreon_credentials_state_enum = PatreonCredentialsStateEnum;

export const $trip2g_graphql_payment_type = PaymentType;

export const $trip2g_graphql_role = Role;

// Generated @exportType declarations

export type $trip2g_graphql_AdminBackgroundQueueJob = NonNullable<NonNullable<NonNullable<AdminBackgroundQueueQuery>['admin']>['backgroundQueue']>['jobs'][0]

// Generated variable type declarations

export type $trip2g_graphql_DisableApiKeyVariables = DisableApiKeyMutationVariables

export type $trip2g_graphql_AdminCreateApiKeyVariables = AdminCreateApiKeyMutationVariables

export type $trip2g_graphql_AdminApiKeyShowQueryVariables = AdminApiKeyShowQueryQueryVariables

export type $trip2g_graphql_AdminAuditLogsVariables = AdminAuditLogsQueryVariables

export type $trip2g_graphql_AdminClearBackgroundQueueVariables = AdminClearBackgroundQueueMutationVariables

export type $trip2g_graphql_AdminStartBackgroundQueueVariables = AdminStartBackgroundQueueMutationVariables

export type $trip2g_graphql_AdminStopBackgroundQueueVariables = AdminStopBackgroundQueueMutationVariables

export type $trip2g_graphql_AdminBackgroundQueueVariables = AdminBackgroundQueueQueryVariables

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

export type $trip2g_graphql_StartTelegramAccountAuthVariables = StartTelegramAccountAuthMutationVariables

export type $trip2g_graphql_CompleteTelegramAccountAuthVariables = CompleteTelegramAccountAuthMutationVariables

export type $trip2g_graphql_AdminTelegramAccountDialogsVariables = AdminTelegramAccountDialogsQueryVariables

export type $trip2g_graphql_AdminImportTelegramAccountChannelVariables = AdminImportTelegramAccountChannelMutationVariables

export type $trip2g_graphql_AdminTelegramAccountShowDialogsInstantTagsSaveVariables = AdminTelegramAccountShowDialogsInstantTagsSaveMutationVariables

export type $trip2g_graphql_AdminTelegramAccountShowDialogsTagsSaveVariables = AdminTelegramAccountShowDialogsTagsSaveMutationVariables

export type $trip2g_graphql_AdminSignOutTelegramAccountVariables = AdminSignOutTelegramAccountMutationVariables

export type $trip2g_graphql_AdminTelegramAccountUpdateVariables = AdminTelegramAccountUpdateQueryVariables

export type $trip2g_graphql_AdminUpdateTelegramAccountMutationVariables = AdminUpdateTelegramAccountMutationMutationVariables

export type $trip2g_graphql_AdminResetTelegramPublishNoteVariables = AdminResetTelegramPublishNoteMutationVariables

export type $trip2g_graphql_AdminSendTelegramPublishNoteNowVariables = AdminSendTelegramPublishNoteNowMutationVariables

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