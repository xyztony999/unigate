# UniGate integration configuration

All notification delivery is disabled by default. No credentials are committed. Provider credentials stored through the administrator API are encrypted; deployment tokens may instead be supplied through a protected configuration source. A missing capability returns `PROVIDER_UNAVAILABLE`, rather than reporting a successful connection.

## Feishu and Tencent Docs

For each administrator-created connection UUID, configure `unigate.providers.connections.<uuid>.resources` as a comma-separated allowlist. Feishu resource identifiers have `appToken/tableId` form; Tencent spreadsheet identifiers have `fileId/sheetId` form. The HTTP adapter uses only the official `open.feishu.cn`, `accounts.feishu.cn`, and `docs.qq.com` hosts and does not follow HTTP redirects.

For browser OAuth, set that connection's `redirect-uri` to the exact registered HTTPS `/api/v1/integration/connection/callback` URL. Feishu also requires a `scope` containing the reviewed read permissions. Save the client ID and client secret through the administrator API, then authorize from that administrator's browser. State is valid for five minutes, bound to the originating administrator session, and consumed before token exchange. A failed exchange requires a new authorization flow. Expired tokens require reauthorization; automatic refresh is not implemented.

For a deployment-managed token, set `access-token` for that connection; Tencent also needs `open-id` and the connection's client ID. These secrets belong in protected deployment configuration. Token validation and real resource permission checks occur on the provider request. The adapter does not claim that a personal account can access every application type.

Feishu reads questionnaire data from its granted Bitable table, including stable provider record IDs. Tencent reads an explicitly granted result spreadsheet and requires `record-id-column` naming a unique, nonempty, stable response-ID column. Movable spreadsheet row numbers are never used as person identities. It does not silently generate a Tencent collection-form result sheet; configure an already approved result spreadsheet. Read ranges obey 1000 rows, 200 columns, and 10000 cells per call. The pull API durably queues a PULL job and immediately returns 202. A leased worker performs provider HTTP reads outside database transactions, then rechecks event phase, source/connection versions and the original organizer authorization before staging a batch. Automatic failures use five attempts with 1/2/4/8-minute backoff. Pulls do not allocate places.

Official references: [Feishu records](https://open.feishu.cn/document/server-docs/docs/bitable-v1/app-table-record/list), [Feishu OAuth tokens](https://open.feishu.cn/document/authentication-management/access-token/get-user-access-token), [Tencent authorization](https://docs.qq.com/open/document/app/oauth2/authorize.html), [Tencent token exchange](https://docs.qq.com/open/document/app/oauth2/access_token.html), [Tencent range reads](https://docs.qq.com/open/document/app/openapi/v3/sheet/get/get_range.html). No live platform account has been authorized in development tests.

## SMTP email

Explicitly enable `unigate.notifications.email-enabled=true`, configure `unigate.notifications.from`, `unigate.public-base-url`, and Spring's `spring.mail.host`, `port`, `username`, and `password`. SMTP authentication and STARTTLS use `spring.mail.properties.mail.smtp.auth` and `mail.smtp.starttls.enable`. Local SMTP capture tests may set `unigate.notifications.allow-local-test=true`; this permits an HTTP local link in test email only. The production public base URL must use HTTPS.

The scheduled worker claims durable encrypted EMAIL jobs using a 60-second lease in a short transaction, checks expiry, loads current private access credentials, sends outside the database transaction, and finishes with a version compare-and-set in another short transaction. Expired leases can be reclaimed. Automatic failures use 1/2/4/8-minute backoff with five attempts per cycle; an explicitly authorized manual retry starts a new cycle. SMTP acceptance does not prove inbox delivery. SMTP does not offer application-level exactly-once delivery: after an ambiguous server failure, a manually retried job can deliver a duplicate email. Message failure never revokes an approved group's eligibility.

## SMS gateway

An optional deployment-owned JSON gateway can be configured with `unigate.notifications.sms-enabled=true`, `sms-endpoint`, `sms-allowed-hosts`, and `sms-token`. Only an HTTPS endpoint on an explicitly configured host is called; its address cannot come from API requests, and redirects are disabled. The gateway accepts a bearer-authenticated POST containing `phone`, `code`, and `expiresInSeconds`; non-2xx responses fail delivery. This adapter is a deployment integration contract, rather than a claim of built-in Aliyun/Tencent Cloud signing support.

Challenges expire after five minutes, have a 60-second send interval, and allow at most five verification attempts. Rejected attempts commit independently of a rolled-back business operation. Recovery replies do not reveal whether a reference/contact matched. Successful recovery rotates only query/change capabilities. Phone proofs bind the event, original anonymous context, phone, expiry, and one-use challenge.

## Managed files and lifecycle

Set `unigate.storage-directory` to a restricted directory outside the public web root. Export files are encrypted before writing; requests cannot choose their paths. Download rechecks the original creator, current event/EXPORT/field grants, expiry, and artifact digest. CSV/XLSX exports neutralize spreadsheet formulas. XLSX uses the first nonempty sheet and ignores later sheets. Formula cells are never evaluated: their row is marked ERROR in preview, while valid rows remain selectable. Macros, malformed headers, excessive rows/columns, and oversized compressed content are rejected.

The lifecycle worker archives due events, saves anonymous counts, deletes linkable live facts in dependency order, removes managed files, and revokes the external per-event key. It records a deletion ledger. Backup coverage remains explicitly `FAILED/BACKUP_COVERAGE_UNVERIFIED` until deployment-level evidence is available; a database deletion alone is not reported as complete cleanup. External downloaded copies remain the recipient's responsibility. Failed managed-file deletion retains the file reference for retry.
