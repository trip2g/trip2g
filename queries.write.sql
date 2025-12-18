-- name: InsertNotePath :one
insert into note_paths (value, value_hash, latest_content_hash)
values (?, ?, ?)
on conflict(value) do update set value = excluded.value
returning id, version_count, latest_content_hash;

-- name: IncrementNoteVersionCount :one
update note_paths
   set version_count = version_count + 1
     , latest_content_hash = ?
 where id = ?
returning version_count;

-- name: InsertNoteVersion :exec
insert into note_versions (path_id, version, content)
values (?, ?, ?);

-- name: InsertUserWithEmail :one
insert into users (email, created_via) values (lower(sqlc.arg(email)), sqlc.arg(created_via))
returning *;

-- name: InsertUserWithTgUserID :one
insert into users (tg_user_id, created_via)
values (?, 'telegram')
returning *;

-- name: InsertSignInCode :exec
insert into sign_in_codes (user_id, code)
values (?, ?);

-- name: DeleteSignInCodesByUserID :exec
delete from sign_in_codes
 where user_id = ?;

-- name: DeleteOffer :one
update offers
   set ends_at = datetime('now')
 where id = ?
returning *;

-- name: InsertSubgraph :exec
insert into subgraphs (name)
values (?)
on conflict(name) do update set hidden = false;

-- name: UpdateAdminSubgraph :one
update subgraphs
   set color = ?, hidden = ?, show_unsubgraph_notes_for_paid_users = ?
 where id = ?
returning *;

-- name: CreateUserSubgraphAccess :one
insert into user_subgraph_accesses (user_id, subgraph_id, purchase_id, expires_at)
values (?, ?, ?, ?)
returning *;

-- name: UpdateUserSubgraphAccess :one
update user_subgraph_accesses
   set expires_at = ?
     , subgraph_id = ?
 where id = ?
returning *;

-- name: CreateRevoke :one
insert into revokes (target_type, target_id, by_id, reason)
values (?, ?, ?, ?)
returning id;

-- name: RevokeUserSubgraphAccess :exec
update user_subgraph_accesses
   set revoke_id = ?
 where id = ?;

-- name: BanUser :exec
insert into user_bans (user_id, banned_by, reason)
values (?, ?, ?);

-- name: UnbanUser :exec
delete from user_bans where user_id = ?;

-- name: InsertUserNoteView :exec
insert into user_note_views (user_id, version_id, referer_version_id) values (?, ?, ?);

-- name: UpsertUserNoteDailyView :one
-- Unfortunately, sqlc cannot generate a parameter for greatest(count + 1, sqlc.arg(max_count)).
insert into user_note_daily_view_counts (user_id, path_id) values (?, ?)
on conflict(user_id, path_id) do update set count = count + 1
returning count;

-- name: IncreaseUserNoteViewCount :exec
update users
   set note_view_count = note_view_count + 1
 where id = ?;

-- name: InsertPurchase :exec
insert into purchases (id, email, offer_id, payment_provider, payment_data, price_usd, status)
values (?, ?, ?, ?, ?, ?, ?);

-- name: UpdatePurchaseStatus :exec
update purchases
   set status = ?
     , payment_data = ?
 where id = ?;

-- name: InsertNoteAsset :one
insert into note_assets (absolute_path, file_name, sha256_hash, size)
values (?, ?, ?, ?)
returning *;

-- name: UpsertNoteVersionAsset :exec
insert into note_version_assets (asset_id, version_id, path)
values (?, ?, ?)
on conflict (asset_id, version_id, path) do update set created_at = datetime('now');

-- name: InsertAcmeCert :exec
insert into acme_certs (key, value)
values (?, ?);

-- name: DeleteAcmeCert :exec
delete from acme_certs where key = ?;

-- name: InsertAPIKey :one
insert into api_keys (value, created_by, description)
values (?, ?, ?)
returning *;

-- name: DisableApiKey :one
update api_keys
  set disabled_by = ?, disabled_at = datetime('now')
 where id = ?
returning *;

-- name: InsertGitToken :one
insert into git_tokens (value_sha256, admin_id, description, can_pull, can_push)
values (?, ?, ?, ?, ?)
returning *;

-- name: DisableGitToken :one
update git_tokens
  set disabled_by = ?, disabled_at = datetime('now')
 where id = ?
returning *;

-- name: InsertAPIKeyLog :exec
insert into api_key_logs (api_key_id, ip_id, action_id)
values (?,
  (select id from api_key_log_ips where value = sqlc.arg(ip)),
  (select id from api_key_log_actions where name = sqlc.arg(action)));

-- name: UpsertAPIKeyLogAction :exec
insert into api_key_log_actions (name)
values (?)
on conflict(name) do nothing;

-- name: UpsertAPIKeyLogIP :exec
insert into api_key_log_ips (value)
values (?)
on conflict(value) do nothing;

-- name: InsertRelease :one
insert into releases (created_by, title, home_note_version_id, is_live)
values (?, ?, ?, ?)
returning *;

-- name: InsertReleaseNoteVersion :exec
insert into release_note_versions (release_id, note_version_id)
values (?, ?);

-- name: ChangeLiveRelease :exec
update releases set is_live = (sqlc.arg(id) = id);

-- name: UpdateNoteGraphPositionByPathID :exec
update note_paths
   set graph_position_x = ?
     , graph_position_y = ?
 where id = ?;

-- name: InsertOffer :one
insert into offers (public_id, lifetime, price_usd, starts_at, ends_at)
values (?, ?, ?, ?, ?)
returning *;

-- name: InsertOfferSubgraph :exec
insert into offer_subgraphs (offer_id, subgraph_id)
values (?, ?);

-- name: UpdateOffer :one
update offers
   set lifetime = coalesce(sqlc.narg(lifetime), lifetime)
     , price_usd = coalesce(sqlc.narg(price_usd), price_usd)
     , starts_at = coalesce(sqlc.narg(starts_at), starts_at)
     , ends_at = coalesce(sqlc.narg(ends_at), ends_at)
 where id = sqlc.arg(id)
returning *;

-- name: DeleteOfferSubgraphs :exec
delete from offer_subgraphs where offer_id = ?;

-- name: InsertAdmin :one
insert into admins (user_id, granted_by)
values (?, ?)
returning *;

-- name: HideNotePath :exec
update note_paths
   set hidden_by = ?
     , hidden_at = datetime('now')
 where value = ?;

-- name: UnhideNotePath :exec
update note_paths
   set hidden_by = null
     , hidden_at = null
 where value = ?;

-- name: InsertRedirect :one
insert into redirects (created_by, pattern, ignore_case, is_regex, target)
values (?, ?, ?, ?, ?)
returning *;

-- name: UpdateRedirect :one
update redirects
   set pattern = ?
     , ignore_case = ?
     , is_regex = ?
     , target = ?
 where id = ?
returning *;

-- name: DeleteRedirect :exec
delete from redirects where id = ?;

-- name: UpsertNotFoundHit :exec
insert into not_found_paths (path)
values (?)
on conflict (path) do update set total_hits = total_hits + 1, last_hit_at = datetime('now');

-- name: UpsertNotFoundIPHit :exec
insert into not_found_ip_hits (ip, total_hits)
values (?, ?)
on conflict(ip) do
update set total_hits = excluded.total_hits, last_hit_at = datetime('now');

-- name: InsertNotFoundIgnoredPattern :one
insert into not_found_ignored_patterns (pattern, created_by)
values (?, ?)
returning *;

-- name: UpdateNotFoundIgnoredPattern :one
update not_found_ignored_patterns
set pattern = ?
where id = ?
returning *;

-- name: DeleteNotFoundIgnoredPattern :exec
delete from not_found_ignored_patterns where id = ?;

-- name: ResetNotFoundPathTotalHits :one
update not_found_paths
set total_hits = 1, last_hit_at = datetime('now')
where id = ?
returning *;

-- name: InsertTgUserProfile :exec
insert into tg_user_profiles (chat_id, bot_id, first_name, last_name, username, sha256_hash)
values (?, ?, ?, ?, ?, ?)
on conflict(sha256_hash) do nothing;

-- name: UpsertTgUserState :exec
insert into tg_user_states (chat_id, bot_id, value, data, update_count)
values (?, ?, ?, ?, ?)
on conflict(chat_id, bot_id) do update set
  value = excluded.value,
  data = excluded.data,
  update_count = excluded.update_count,
  updated_at = current_timestamp;

-- name: UpsertTgBotChat :exec
insert into tg_bot_chats (telegram_id, chat_type, chat_title, can_invite, bot_id)
values (?, ?, ?, ?, ?)
on conflict(telegram_id) do update set
  chat_type = excluded.chat_type,
  chat_title = excluded.chat_title,
  can_invite = excluded.can_invite,
  removed_at = null;

-- name: MarkTgBotChatRemoved :exec
update tg_bot_chats
set removed_at = current_timestamp
where telegram_id = ?;

-- name: UpdateTgBotChatCanInvite :exec
update tg_bot_chats
set can_invite = ?
where telegram_id = ?;

-- name: InsertTgChatMember :exec
insert into tg_chat_members (user_id, chat_id)
values (?, ?)
on conflict(user_id, chat_id) do nothing;

-- name: RemoveTgChatMember :exec
delete from tg_chat_members
where user_id = ? and chat_id = ?;

-- name: InsertTgBot :one
insert into tg_bots (token, name, description, created_by)
values (?, ?, ?, ?)
returning *;

-- name: UpdateTgBot :one
update tg_bots
set description = coalesce(sqlc.narg(description), description),
    enabled = coalesce(sqlc.narg(enabled), enabled)
where id = sqlc.arg(id)
returning *;

-- name: InsertTgChatSubgraphAccess :one
insert into tg_chat_subgraph_accesses (chat_id, subgraph_id)
values (?, ?)
returning *;

-- name: DeleteTgChatSubgraphAccess :exec
delete from tg_chat_subgraph_accesses
where id = ?;

-- name: DeleteTgChatSubgraphAccessesByChatID :exec
delete from tg_chat_subgraph_accesses
where chat_id = ?;

-- name: InsertTgChatSubgraphInvite :one
insert into tg_bot_chat_subgraph_invites (chat_id, subgraph_id, created_by)
values (?, ?, ?)
on conflict (chat_id, subgraph_id) do update set created_at = current_timestamp
returning *;

-- name: DeleteTgChatSubgraphInvitesByChatID :exec
delete from tg_bot_chat_subgraph_invites
where chat_id = ?;

-- name: InsertWaitListEmailRequest :exec
insert into wait_list_email_requests (email, note_path_id, ip)
values (?, ?, ?);

-- name: InsertWaitListTgBotRequest :exec
insert into wait_list_tg_bot_requests (bot_id, chat_id, note_path_id)
values (?, ?, ?);

-- name: InsertPatreonCredentials :one
insert into patreon_credentials (created_by, creator_access_token)
values (?, ?)
returning *;

-- name: SoftDeletePatreonCredentials :one
update patreon_credentials
set deleted_at = current_timestamp, deleted_by = ?
where id = ? and deleted_at is null
returning *;

-- name: RestorePatreonCredentials :one
update patreon_credentials
set deleted_at = null, deleted_by = null
where id = ? and deleted_at is not null
returning *;

-- name: InsertPatreonCampaign :exec
insert into patreon_campaigns (credentials_id, campaign_id, attributes)
values (?, ?, ?);

-- name: UpsertPatreonCampaign :exec
insert into patreon_campaigns (credentials_id, campaign_id, attributes)
values (?, ?, ?)
on conflict(credentials_id, campaign_id) do update set
  attributes = excluded.attributes,
  missed_at = null;

-- name: UpsertPatreonTier :exec
insert into patreon_tiers (campaign_id, tier_id, title, amount_cents, attributes)
values (?, ?, ?, ?, ?)
on conflict(campaign_id, tier_id) do update set
  title = excluded.title,
  amount_cents = excluded.amount_cents,
  attributes = excluded.attributes,
  missed_at = null;

-- name: UpsertPatreonMember :exec
insert into patreon_members (patreon_id, campaign_id, status, email)
values (?, ?, ?, ?)
on conflict(patreon_id, campaign_id) do update set
  status = excluded.status,
  email = excluded.email;

-- name: UpdatePatreonMemberUserID :exec
update patreon_members
   set user_id = ?
 where id = ?;

-- name: UpdatePatreonCredentialsSyncedAt :exec
update patreon_credentials
set synced_at = current_timestamp
where id = ?;

-- name: UpdatePatreonCredentialsWebhookSecret :exec
update patreon_credentials
set webhook_secret = ?
where id = ?;

-- name: ClearPatreonCredentialsWebhookSecret :exec
update patreon_credentials
set webhook_secret = null
where id = ?;

-- name: InsertBoostyCredentials :one
insert into boosty_credentials (created_by, auth_data, device_id, blog_name)
values (?, ?, ?, ?)
returning *;

-- name: SoftDeleteBoostyCredentials :one
update boosty_credentials
set deleted_at = current_timestamp, deleted_by = ?
where id = ? and deleted_at is null
returning *;

-- name: RestoreBoostyCredentials :one
update boosty_credentials
set deleted_at = null, deleted_by = null
where id = ? and deleted_at is not null
returning *;

-- name: UpdateBoostyCredentials :one
update boosty_credentials
set auth_data = ?, device_id = ?, blog_name = ?
where id = ?
returning *;

-- name: UpdateBoostyCredentialsTokens :one
update boosty_credentials
set auth_data = ?, expires_at = ?
where id = ?
returning *;

-- name: UpdateBoostyCredentialsSyncedAt :exec
update boosty_credentials
set synced_at = current_timestamp
where id = ?;

-- name: InsertBoostyTier :exec
insert into boosty_tiers (credentials_id, boosty_id, name, data)
values (?, ?, ?, ?);

-- name: UpsertBoostyTier :exec
insert into boosty_tiers (credentials_id, boosty_id, name, data)
values (?, ?, ?, ?)
on conflict(credentials_id, boosty_id) do update set
  name = excluded.name,
  data = excluded.data,
  missed_at = null;

-- name: DeleteBoostyTierSubgraphsByTierID :exec
delete from boosty_tier_subgraphs where tier_id = ?;

-- name: InsertBoostyTierSubgraph :exec
insert into boosty_tier_subgraphs (tier_id, subgraph_id, created_by)
values (?, ?, ?);

-- name: InsertBoostyMember :exec
insert into boosty_members (boosty_id, email, status, data)
values (?, ?, ?, ?);

-- name: UpsertBoostyMember :exec
insert into boosty_members (credentials_id, boosty_id, email, status, data, current_tier_id)
values (?, ?, ?, ?, ?, ?)
on conflict(credentials_id, boosty_id) do update set
  email = excluded.email,
  status = excluded.status,
  data = excluded.data,
  current_tier_id = excluded.current_tier_id,
  missed_at = null;

-- name: UpdateBoostyMemberUserID :exec
update boosty_members
set user_id = ?
where id = ?;

-- name: MarkBoostyMembersAsMissed :exec
update boosty_members
set missed_at = datetime('now')
where missed_at is null
  and boosty_id not in (sqlc.slice('boosty_ids'));

-- name: MarkBoostyTiersAsMissed :exec
update boosty_tiers
set missed_at = datetime('now')
where credentials_id = ?
  and missed_at is null
  and boosty_id not in (sqlc.slice('boosty_ids'));

-- name: MarkPatreonMembersAsMissed :exec
update patreon_members
set status = 'missed'
where campaign_id = ?
  and patreon_id not in (
    select json_extract(value, '$') 
    from json_each(?)
  );

-- name: SetPatreonMemberCurrentTier :exec
update patreon_members
set current_tier_id = ?
where id = ?;

-- name: DeletePatreonTierSubgraphsByTierID :exec
delete from patreon_tier_subgraphs
where tier_id = ?;

-- name: InsertPatreonTierSubgraph :exec
insert into patreon_tier_subgraphs (tier_id, subgraph_id, created_by)
values (?, ?, ?);

-- name: InsertUserFavoriteNote :exec
insert into user_favorite_notes (user_id, note_version_id)
values (?, ?) on conflict do nothing;

-- name: DeleteUserFavoriteNote :exec
delete from user_favorite_notes
where user_id = ? and note_version_id = ?;

-- name: InsertTgAttachCode :exec
insert into tg_attach_codes (user_id, bot_id, code)
values (?, ?, ?);

-- name: DeleteTgAttachCode :exec
delete from tg_attach_codes where code = ?;

-- name: DeleteTgAttachCodesByUser :exec
delete from tg_attach_codes where user_id = ?;

-- name: UpdateUserTgID :exec
update users set tg_user_id = ? where id = ?;

-- name: ClearTgUserIDByTgUserID :exec
update users set tg_user_id = null where tg_user_id = ?;

-- name: UpdateUser :one
update users set email = coalesce(sqlc.narg(email), email) where id = sqlc.arg(id) returning *;

-- name: InsertTgBotChatSubgraphAccess :exec
insert into tg_bot_chat_subgraph_accesses (chat_id, user_id, subgraph_id)
values (?, ?, (select id from subgraphs where name = ?))
on conflict (chat_id, user_id, subgraph_id) do update set created_at = current_timestamp;

-- name: UpdateTgBotChatSubgraphAccessJoinedAt :exec
update tg_bot_chat_subgraph_accesses
set joined_at = current_timestamp
where chat_id = ? and user_id = ? and subgraph_id = (select id from subgraphs where name = ?);

-- name: DeleteTgBotChatSubgraphAccess :exec
delete from tg_bot_chat_subgraph_accesses
where chat_id = ? and user_id = ? and subgraph_id = ?;

-- name: InsertAuditLog :exec
insert into audit_logs (level, message, params)
values (?, ?, ?);

-- name: InsertHTMLInjection :one
insert into html_injections (
  description,
  position,
  placement,
  content,
  active_from,
  active_to
) values (?, ?, ?, ?, ?, ?)
returning *;

-- name: UpdateHTMLInjection :one
update html_injections
set description = ?,
    position = ?,
    placement = ?,
    content = ?,
    active_from = ?,
    active_to = ?
where id = ?
returning *;

-- name: DeleteHTMLInjection :exec
delete from html_injections
where id = ?;

-- name: UpsertCronJob :exec
insert into cron_jobs (name, expression)
select ?, ?
 where not exists (select 1 from cron_jobs where name = ?1);

-- name: DeleteCronJobByName :exec
delete from cron_jobs where name = ?;

-- name: InsertCronJobExecution :one
insert into cron_job_executions (job_id, status)
values (?, ?)
returning *;

-- name: UpdateCronJobExecution :one
update cron_job_executions
set finished_at = datetime('now'),
    status = ?,
    report_data = ?,
    error_message = ?
where id = ?
returning *;

-- name: UpdateCronJobLastExec :exec
update cron_jobs
set last_exec_at = datetime('now')
where id = ?;

-- name: UpdateRunningCronJobExecutions :exec
update cron_job_executions
  set status = ?, error_message = ?
where job_id = ?
  and status = 'running';

-- name: UpdateCronJob :one
update cron_jobs
set enabled = ?, expression = ?
where id = ?
returning *;

-- name: DeleteOldCronJobExecutions :execrows
delete from cron_job_executions
where started_at < datetime('now', '-7 days');

-- name: UpdateNotionIntegrationVerificationToken :exec
update notion_integrations
   set verification_token = ?
 where id = ?;

-- name: InsertConfigVersion :one
insert into config_versions (created_by, show_draft_versions, default_layout, timezone, robots_txt)
values (?, ?, ?, ?, ?)
returning *;

-- name: InsertTelegramPublishTags :exec
insert into telegram_publish_tags (label)
values (?)
on conflict(label) do nothing;

-- name: UpsertTelegramPublishNote :exec
insert into telegram_publish_notes (note_path_id, publish_at, error_count)
values (?, ?, ?)
on conflict(note_path_id) do update set
  publish_at = excluded.publish_at,
  error_count = excluded.error_count;

-- name: DeleteTelegramPublishNoteTagsByPathID :exec
delete from telegram_publish_note_tags where note_path_id = ?;

-- name: UpsertTelegramPublishNoteTag :exec
insert into telegram_publish_note_tags (note_path_id, tag_id)
values (?, ?)
on conflict(note_path_id, tag_id) do nothing;

-- name: DeleteTelegramPublishChatsByChatID :exec
delete from telegram_publish_chats where chat_id = ?;

-- name: InsertTelegramPublishChat :exec
insert into telegram_publish_chats (chat_id, tag_id, created_by)
values (?, ?, ?);

-- name: DeleteTelegramPublishInstantChatsByChatID :exec
delete from telegram_publish_instant_chats where chat_id = ?;

-- name: InsertTelegramPublishInstantChat :exec
insert into telegram_publish_instant_chats (chat_id, tag_id, created_by)
values (?, ?, ?);

-- name: UpdateTelegramPublishNoteAsPublished :exec
update telegram_publish_notes
   set published_at = datetime('now')
     , published_version_id = ?
 where note_path_id = ?;

-- name: InsertTelegramPublishSentMessage :exec
insert into telegram_publish_sent_messages (note_path_id, chat_id, message_id, instant, content_hash, content, post_type)
values (?, ?, ?, ?, ?, ?, ?);

-- name: ResetTelegramPublishNote :exec
update telegram_publish_notes
   set published_at = null
     , published_version_id = null
 where note_path_id = ?;

-- name: DeleteTelegramPublishSentMessagesByNotePathID :exec
delete from telegram_publish_sent_messages where note_path_id = ?;

-- name: UpdateTelegramPublishSentMessageContent :exec
update telegram_publish_sent_messages
   set content_hash = ?
     , content = ?
 where note_path_id = ?
   and chat_id = ?
   and message_id = ?;



-- name: DeleteGoqiteJobsByQueue :execresult
delete from goqite where queue = ?;


-- name: DeleteNoteAsset :exec
delete from note_assets where id = ?;

-- name: SetTelegramPublishNoteLastError :exec
update telegram_publish_notes
   set last_error = ?
 where note_path_id = ?;

-- name: ClearTelegramPublishNoteLastError :exec
update telegram_publish_notes
   set last_error = null
 where note_path_id = ?;

-- ============================================
-- Telegram Accounts
-- ============================================

-- name: InsertTelegramAccount :one
insert into telegram_accounts (phone, session_data, display_name, is_premium, api_id, api_hash, created_by)
values (?, ?, ?, ?, ?, ?, ?)
returning *;

-- name: UpdateTelegramAccount :exec
update telegram_accounts
   set display_name = coalesce(sqlc.narg(display_name), display_name)
     , enabled = coalesce(sqlc.narg(enabled), enabled)
     , is_premium = coalesce(sqlc.narg(is_premium), is_premium)
     , session_data = coalesce(sqlc.narg(session_data), session_data)
     , api_id = coalesce(sqlc.narg(api_id), api_id)
     , api_hash = coalesce(sqlc.narg(api_hash), api_hash)
 where id = ?1;

-- name: UpdateTelegramAccountAppConfig :exec
update telegram_accounts
   set app_config = ?
 where id = ?;

-- name: DeleteTelegramPublishAccountChatsByAccountAndChatID :exec
delete from telegram_publish_account_chats
 where account_id = ?
   and telegram_chat_id = ?;

-- name: InsertTelegramPublishAccountChat :exec
insert into telegram_publish_account_chats (account_id, telegram_chat_id, tag_id, created_by)
values (?, ?, ?, ?);

-- name: DeleteTelegramPublishAccountInstantChatsByAccountAndChatID :exec
delete from telegram_publish_account_instant_chats
 where account_id = ?
   and telegram_chat_id = ?;

-- name: InsertTelegramPublishAccountInstantChat :exec
insert into telegram_publish_account_instant_chats (account_id, telegram_chat_id, tag_id, created_by)
values (?, ?, ?, ?);

-- name: InsertTelegramPublishSentAccountMessage :exec
insert into telegram_publish_sent_account_messages
       (note_path_id, account_id, telegram_chat_id, message_id, instant, content_hash, content, post_type)
values (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateTelegramPublishSentAccountMessageContent :exec
update telegram_publish_sent_account_messages
   set content_hash = ?
     , content = ?
     , post_type = ?
 where note_path_id = ?
   and account_id = ?
   and telegram_chat_id = ?
   and message_id = ?;

-- name: DeleteTelegramPublishSentAccountMessagesByNotePathID :exec
delete from telegram_publish_sent_account_messages where note_path_id = ?;

-- name: InsertUncommittedPath :exec
insert into note_uncommitted_paths (note_path_id)
values (?)
on conflict do nothing;

-- name: ClearUncommittedPaths :exec
delete from note_uncommitted_paths;
