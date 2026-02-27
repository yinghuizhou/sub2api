package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/proxy"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type sqlQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type proxyRepository struct {
	client *dbent.Client
	sql    sqlQuerier
	sqlDB  *sql.DB // for transactions
}

func NewProxyRepository(client *dbent.Client, sqlDB *sql.DB) service.ProxyRepository {
	return &proxyRepository{client: client, sql: sqlDB, sqlDB: sqlDB}
}

func newProxyRepositoryWithSQL(client *dbent.Client, sqlq sqlQuerier) *proxyRepository {
	return &proxyRepository{client: client, sql: sqlq}
}

func (r *proxyRepository) Create(ctx context.Context, proxyIn *service.Proxy) error {
	builder := r.client.Proxy.Create().
		SetName(proxyIn.Name).
		SetProtocol(proxyIn.Protocol).
		SetHost(proxyIn.Host).
		SetPort(proxyIn.Port).
		SetStatus(proxyIn.Status).
		SetIsDedicated(proxyIn.IsDedicated).
		SetHealthCheckFailures(proxyIn.HealthCheckFailures)
	if proxyIn.Username != "" {
		builder.SetUsername(proxyIn.Username)
	}
	if proxyIn.Password != "" {
		builder.SetPassword(proxyIn.Password)
	}
	if proxyIn.Region != "" {
		builder.SetRegion(proxyIn.Region)
	}
	if proxyIn.GroupName != "" {
		builder.SetGroupName(proxyIn.GroupName)
	}
	if proxyIn.OvpnConfig != "" {
		builder.SetOvpnConfig(proxyIn.OvpnConfig)
	}
	if proxyIn.OvpnUsername != "" {
		builder.SetOvpnUsername(proxyIn.OvpnUsername)
	}
	if proxyIn.OvpnPassword != "" {
		builder.SetOvpnPassword(proxyIn.OvpnPassword)
	}
	if proxyIn.VpnStatus != "" {
		builder.SetVpnStatus(proxyIn.VpnStatus)
	}
	if proxyIn.VpnExitIP != "" {
		builder.SetVpnExitIP(proxyIn.VpnExitIP)
	}
	if proxyIn.HealthStatus != "" {
		builder.SetHealthStatus(proxyIn.HealthStatus)
	}
	if proxyIn.LatencyMs != nil {
		builder.SetLatencyMs(*proxyIn.LatencyMs)
	}
	if proxyIn.LastHealthAt != nil {
		builder.SetLastHealthAt(*proxyIn.LastHealthAt)
	}

	created, err := builder.Save(ctx)
	if err == nil {
		applyProxyEntityToService(proxyIn, created)
	}
	return err
}

func (r *proxyRepository) GetByID(ctx context.Context, id int64) (*service.Proxy, error) {
	m, err := r.client.Proxy.Get(ctx, id)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, service.ErrProxyNotFound
		}
		return nil, err
	}
	return proxyEntityToService(m), nil
}

func (r *proxyRepository) ListByIDs(ctx context.Context, ids []int64) ([]service.Proxy, error) {
	if len(ids) == 0 {
		return []service.Proxy{}, nil
	}

	proxies, err := r.client.Proxy.Query().
		Where(proxy.IDIn(ids...)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]service.Proxy, 0, len(proxies))
	for i := range proxies {
		out = append(out, *proxyEntityToService(proxies[i]))
	}
	return out, nil
}

func (r *proxyRepository) Update(ctx context.Context, proxyIn *service.Proxy) error {
	builder := r.client.Proxy.UpdateOneID(proxyIn.ID).
		SetName(proxyIn.Name).
		SetProtocol(proxyIn.Protocol).
		SetHost(proxyIn.Host).
		SetPort(proxyIn.Port).
		SetStatus(proxyIn.Status).
		SetIsDedicated(proxyIn.IsDedicated).
		SetHealthCheckFailures(proxyIn.HealthCheckFailures)
	if proxyIn.Username != "" {
		builder.SetUsername(proxyIn.Username)
	} else {
		builder.ClearUsername()
	}
	if proxyIn.Password != "" {
		builder.SetPassword(proxyIn.Password)
	} else {
		builder.ClearPassword()
	}
	if proxyIn.Region != "" {
		builder.SetRegion(proxyIn.Region)
	} else {
		builder.ClearRegion()
	}
	if proxyIn.GroupName != "" {
		builder.SetGroupName(proxyIn.GroupName)
	} else {
		builder.ClearGroupName()
	}
	if proxyIn.OvpnConfig != "" {
		builder.SetOvpnConfig(proxyIn.OvpnConfig)
	} else {
		builder.ClearOvpnConfig()
	}
	if proxyIn.OvpnUsername != "" {
		builder.SetOvpnUsername(proxyIn.OvpnUsername)
	} else {
		builder.ClearOvpnUsername()
	}
	if proxyIn.OvpnPassword != "" {
		builder.SetOvpnPassword(proxyIn.OvpnPassword)
	} else {
		builder.ClearOvpnPassword()
	}
	if proxyIn.VpnStatus != "" {
		builder.SetVpnStatus(proxyIn.VpnStatus)
	} else {
		builder.ClearVpnStatus()
	}
	if proxyIn.VpnExitIP != "" {
		builder.SetVpnExitIP(proxyIn.VpnExitIP)
	} else {
		builder.ClearVpnExitIP()
	}
	if proxyIn.HealthStatus != "" {
		builder.SetHealthStatus(proxyIn.HealthStatus)
	} else {
		builder.ClearHealthStatus()
	}
	if proxyIn.LatencyMs != nil {
		builder.SetLatencyMs(*proxyIn.LatencyMs)
	} else {
		builder.ClearLatencyMs()
	}
	if proxyIn.LastHealthAt != nil {
		builder.SetLastHealthAt(*proxyIn.LastHealthAt)
	} else {
		builder.ClearLastHealthAt()
	}

	updated, err := builder.Save(ctx)
	if err == nil {
		applyProxyEntityToService(proxyIn, updated)
		return nil
	}
	if dbent.IsNotFound(err) {
		return service.ErrProxyNotFound
	}
	return err
}

func (r *proxyRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.client.Proxy.Delete().Where(proxy.IDEQ(id)).Exec(ctx)
	return err
}

func (r *proxyRepository) List(ctx context.Context, params pagination.PaginationParams) ([]service.Proxy, *pagination.PaginationResult, error) {
	return r.ListWithFilters(ctx, params, "", "", "")
}

// ListWithFilters lists proxies with optional filtering by protocol, status, and search query
func (r *proxyRepository) ListWithFilters(ctx context.Context, params pagination.PaginationParams, protocol, status, search string) ([]service.Proxy, *pagination.PaginationResult, error) {
	q := r.client.Proxy.Query()
	if protocol != "" {
		q = q.Where(proxy.ProtocolEQ(protocol))
	}
	if status != "" {
		q = q.Where(proxy.StatusEQ(status))
	}
	if search != "" {
		q = q.Where(proxy.NameContainsFold(search))
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	proxies, err := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Desc(proxy.FieldID)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	outProxies := make([]service.Proxy, 0, len(proxies))
	for i := range proxies {
		outProxies = append(outProxies, *proxyEntityToService(proxies[i]))
	}

	return outProxies, paginationResultFromTotal(int64(total), params), nil
}

// ListWithFiltersAndAccountCount lists proxies with filters and includes account count per proxy
func (r *proxyRepository) ListWithFiltersAndAccountCount(ctx context.Context, params pagination.PaginationParams, protocol, status, search, groupName string) ([]service.ProxyWithAccountCount, *pagination.PaginationResult, error) {
	q := r.client.Proxy.Query()
	if protocol != "" {
		q = q.Where(proxy.ProtocolEQ(protocol))
	}
	if status != "" {
		q = q.Where(proxy.StatusEQ(status))
	}
	if search != "" {
		q = q.Where(proxy.NameContainsFold(search))
	}
	if groupName != "" {
		q = q.Where(proxy.GroupNameEQ(groupName))
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	proxies, err := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Desc(proxy.FieldID)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Collect IDs of the returned page, then fetch counts only for those proxies
	ids := make([]int64, 0, len(proxies))
	for _, p := range proxies {
		ids = append(ids, p.ID)
	}
	counts, err := r.GetAccountCountsForProxyIDs(ctx, ids)
	if err != nil {
		return nil, nil, err
	}

	// Build result with account counts
	result := make([]service.ProxyWithAccountCount, 0, len(proxies))
	for i := range proxies {
		proxyOut := proxyEntityToService(proxies[i])
		if proxyOut == nil {
			continue
		}
		result = append(result, service.ProxyWithAccountCount{
			Proxy:        *proxyOut,
			AccountCount: counts[proxyOut.ID],
		})
	}

	return result, paginationResultFromTotal(int64(total), params), nil
}

func (r *proxyRepository) ListActive(ctx context.Context) ([]service.Proxy, error) {
	proxies, err := r.client.Proxy.Query().
		Where(proxy.StatusEQ(service.StatusActive)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	outProxies := make([]service.Proxy, 0, len(proxies))
	for i := range proxies {
		outProxies = append(outProxies, *proxyEntityToService(proxies[i]))
	}
	return outProxies, nil
}

// ExistsByHostPortAuth checks if a proxy with the same host, port, username, and password exists
func (r *proxyRepository) ExistsByHostPortAuth(ctx context.Context, host string, port int, username, password string) (bool, error) {
	q := r.client.Proxy.Query().
		Where(proxy.HostEQ(host), proxy.PortEQ(port))

	if username == "" {
		q = q.Where(proxy.Or(proxy.UsernameIsNil(), proxy.UsernameEQ("")))
	} else {
		q = q.Where(proxy.UsernameEQ(username))
	}
	if password == "" {
		q = q.Where(proxy.Or(proxy.PasswordIsNil(), proxy.PasswordEQ("")))
	} else {
		q = q.Where(proxy.PasswordEQ(password))
	}

	count, err := q.Count(ctx)
	return count > 0, err
}

// CountAccountsByProxyID returns the number of accounts using a specific proxy
func (r *proxyRepository) CountAccountsByProxyID(ctx context.Context, proxyID int64) (int64, error) {
	var count int64
	if err := scanSingleRow(ctx, r.sql, "SELECT COUNT(*) FROM accounts WHERE proxy_id = $1 AND deleted_at IS NULL", []any{proxyID}, &count); err != nil {
		return 0, err
	}
	return count, nil
}

// CountAccountsByGroupName returns a map of proxy_group to account count.
func (r *proxyRepository) CountAccountsByGroupName(ctx context.Context) (map[string]int64, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT proxy_group, COUNT(*) as cnt
		FROM accounts
		WHERE proxy_group IS NOT NULL
		  AND proxy_group != ''
		  AND deleted_at IS NULL
		GROUP BY proxy_group
	`)
	if err != nil {
		return nil, fmt.Errorf("count accounts by group: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]int64)
	for rows.Next() {
		var group string
		var count int64
		if err := rows.Scan(&group, &count); err != nil {
			return nil, fmt.Errorf("scan group count: %w", err)
		}
		result[group] = count
	}
	return result, rows.Err()
}

func (r *proxyRepository) ListAccountSummariesByProxyID(ctx context.Context, proxyID int64) ([]service.ProxyAccountSummary, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT id, name, platform, type, notes
		FROM accounts
		WHERE proxy_id = $1 AND deleted_at IS NULL
		ORDER BY id DESC
	`, proxyID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.ProxyAccountSummary, 0)
	for rows.Next() {
		var (
			id       int64
			name     string
			platform string
			accType  string
			notes    sql.NullString
		)
		if err := rows.Scan(&id, &name, &platform, &accType, &notes); err != nil {
			return nil, err
		}
		var notesPtr *string
		if notes.Valid {
			notesPtr = &notes.String
		}
		out = append(out, service.ProxyAccountSummary{
			ID:       id,
			Name:     name,
			Platform: platform,
			Type:     accType,
			Notes:    notesPtr,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// GetAccountCountsForProxies returns a map of proxy ID to account count for all proxies
func (r *proxyRepository) GetAccountCountsForProxies(ctx context.Context) (counts map[int64]int64, err error) {
	rows, err := r.sql.QueryContext(ctx, "SELECT proxy_id, COUNT(*) AS count FROM accounts WHERE proxy_id IS NOT NULL AND deleted_at IS NULL GROUP BY proxy_id")
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			counts = nil
		}
	}()

	counts = make(map[int64]int64)
	for rows.Next() {
		var proxyID, count int64
		if err = rows.Scan(&proxyID, &count); err != nil {
			return nil, err
		}
		counts[proxyID] = count
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return counts, nil
}

// GetAccountCountsForProxyIDs returns a map of proxy ID to account count for the given set of proxy IDs.
// Unlike GetAccountCountsForProxies, this only scans rows for the requested IDs.
func (r *proxyRepository) GetAccountCountsForProxyIDs(ctx context.Context, ids []int64) (counts map[int64]int64, err error) {
	if len(ids) == 0 {
		return make(map[int64]int64), nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	q := fmt.Sprintf(
		"SELECT proxy_id, COUNT(*) AS count FROM accounts WHERE proxy_id IN (%s) AND deleted_at IS NULL GROUP BY proxy_id",
		strings.Join(placeholders, ","),
	)
	rows, err := r.sql.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			counts = nil
		}
	}()
	counts = make(map[int64]int64, len(ids))
	for rows.Next() {
		var proxyID, count int64
		if err = rows.Scan(&proxyID, &count); err != nil {
			return nil, err
		}
		counts[proxyID] = count
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return counts, nil
}

// ListActiveWithAccountCount returns all active proxies with account count, sorted by creation time descending
func (r *proxyRepository) ListActiveWithAccountCount(ctx context.Context) ([]service.ProxyWithAccountCount, error) {
	proxies, err := r.client.Proxy.Query().
		Where(proxy.StatusEQ(service.StatusActive)).
		Order(dbent.Desc(proxy.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	// Get account counts
	counts, err := r.GetAccountCountsForProxies(ctx)
	if err != nil {
		return nil, err
	}

	// Build result with account counts
	result := make([]service.ProxyWithAccountCount, 0, len(proxies))
	for i := range proxies {
		proxyOut := proxyEntityToService(proxies[i])
		if proxyOut == nil {
			continue
		}
		result = append(result, service.ProxyWithAccountCount{
			Proxy:        *proxyOut,
			AccountCount: counts[proxyOut.ID],
		})
	}

	return result, nil
}

func proxyEntityToService(m *dbent.Proxy) *service.Proxy {
	if m == nil {
		return nil
	}
	out := &service.Proxy{
		ID:                  m.ID,
		Name:                m.Name,
		Protocol:            m.Protocol,
		Host:                m.Host,
		Port:                m.Port,
		Status:              m.Status,
		IsDedicated:         m.IsDedicated,
		HealthCheckFailures: m.HealthCheckFailures,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
	}
	if m.Username != nil {
		out.Username = *m.Username
	}
	if m.Password != nil {
		out.Password = *m.Password
	}
	if m.Region != nil {
		out.Region = *m.Region
	}
	if m.GroupName != nil {
		out.GroupName = *m.GroupName
	}
	if m.OvpnConfig != nil {
		out.OvpnConfig = *m.OvpnConfig
	}
	if m.OvpnUsername != nil {
		out.OvpnUsername = *m.OvpnUsername
	}
	if m.OvpnPassword != nil {
		out.OvpnPassword = *m.OvpnPassword
	}
	if m.VpnStatus != nil {
		out.VpnStatus = *m.VpnStatus
	}
	if m.VpnExitIP != nil {
		out.VpnExitIP = *m.VpnExitIP
	}
	if m.HealthStatus != nil {
		out.HealthStatus = *m.HealthStatus
	}
	if m.LatencyMs != nil {
		out.LatencyMs = m.LatencyMs
	}
	if m.LastHealthAt != nil {
		out.LastHealthAt = m.LastHealthAt
	}
	return out
}

func applyProxyEntityToService(dst *service.Proxy, src *dbent.Proxy) {
	if dst == nil || src == nil {
		return
	}
	dst.ID = src.ID
	dst.CreatedAt = src.CreatedAt
	dst.UpdatedAt = src.UpdatedAt
}

// ListByGroupName returns all active proxies in a given group
func (r *proxyRepository) ListByGroupName(ctx context.Context, groupName string) ([]service.Proxy, error) {
	proxies, err := r.client.Proxy.Query().
		Where(
			proxy.GroupNameEQ(groupName),
			proxy.StatusEQ(service.StatusActive),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]service.Proxy, 0, len(proxies))
	for i := range proxies {
		out = append(out, *proxyEntityToService(proxies[i]))
	}
	return out, nil
}

// ListGroupNames returns all distinct non-null group names
func (r *proxyRepository) ListGroupNames(ctx context.Context) ([]string, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT DISTINCT group_name FROM proxies
		WHERE group_name IS NOT NULL AND group_name != '' AND deleted_at IS NULL
		ORDER BY group_name
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// ListAccountsByProxyGroup returns basic info about accounts using a given proxy_group.
func (r *proxyRepository) ListAccountsByProxyGroup(ctx context.Context, groupName string) ([]service.ProxyAccountSummary, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT id, name, platform, type, notes
		FROM accounts
		WHERE proxy_group = $1 AND deleted_at IS NULL
		ORDER BY id
	`, groupName)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []service.ProxyAccountSummary
	for rows.Next() {
		var (
			id       int64
			name     string
			platform string
			accType  string
			notes    sql.NullString
		)
		if err := rows.Scan(&id, &name, &platform, &accType, &notes); err != nil {
			return nil, err
		}
		var notesPtr *string
		if notes.Valid {
			notesPtr = &notes.String
		}
		out = append(out, service.ProxyAccountSummary{
			ID:       id,
			Name:     name,
			Platform: platform,
			Type:     accType,
			Notes:    notesPtr,
		})
	}
	return out, rows.Err()
}

// --- Proxy Assignment methods (persistent account-proxy binding) ---

func (r *proxyRepository) GetAssignment(ctx context.Context, accountID int64) (*service.ProxyAssignment, error) {
	var a service.ProxyAssignment
	err := scanSingleRow(ctx, r.sql, `
		SELECT id, account_id, proxy_id, proxy_group, assigned_at, assigned_by
		FROM account_proxy_assignments WHERE account_id = $1
	`, []any{accountID}, &a.ID, &a.AccountID, &a.ProxyID, &a.ProxyGroup, &a.AssignedAt, &a.AssignedBy)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *proxyRepository) SetAssignment(ctx context.Context, accountID, proxyID int64, groupName, assignedBy string) error {
	// Use DO NOTHING for auto-assignments to let the first writer win (prevents TOCTOU race).
	// Use DO UPDATE for admin-assignments to allow explicit overrides.
	if assignedBy == "auto" {
		res, err := r.sql.ExecContext(ctx, `
			INSERT INTO account_proxy_assignments (account_id, proxy_id, proxy_group, assigned_by)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (account_id) DO NOTHING
		`, accountID, proxyID, groupName, assignedBy)
		if err != nil {
			return err
		}
		if rows, _ := res.RowsAffected(); rows == 0 {
			// Already assigned by another goroutine — this is expected, not an error
			return nil
		}
		return nil
	}
	_, err := r.sql.ExecContext(ctx, `
		INSERT INTO account_proxy_assignments (account_id, proxy_id, proxy_group, assigned_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (account_id) DO UPDATE SET proxy_id = $2, proxy_group = $3, assigned_by = $4, assigned_at = NOW()
	`, accountID, proxyID, groupName, assignedBy)
	return err
}

func (r *proxyRepository) DeleteAssignment(ctx context.Context, accountID int64) error {
	_, err := r.sql.ExecContext(ctx, `DELETE FROM account_proxy_assignments WHERE account_id = $1`, accountID)
	return err
}

func (r *proxyRepository) ListAssignmentsByGroup(ctx context.Context, groupName string) ([]service.ProxyAssignment, error) {
	return r.ListAssignmentsByGroupPaginated(ctx, groupName, 0, 0)
}

func (r *proxyRepository) ListAssignmentsByGroupPaginated(ctx context.Context, groupName string, offset, limit int) ([]service.ProxyAssignment, error) {
	query := `SELECT id, account_id, proxy_id, proxy_group, assigned_at, assigned_by
		FROM account_proxy_assignments WHERE proxy_group = $1 ORDER BY id`
	var args []any
	args = append(args, groupName)
	if limit > 0 {
		limitIdx := len(args) + 1
		offsetIdx := len(args) + 2
		args = append(args, limit, offset)
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", limitIdx, offsetIdx)
	}
	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []service.ProxyAssignment
	for rows.Next() {
		var a service.ProxyAssignment
		if err := rows.Scan(&a.ID, &a.AccountID, &a.ProxyID, &a.ProxyGroup, &a.AssignedAt, &a.AssignedBy); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *proxyRepository) CountAssignmentsByGroup(ctx context.Context, groupName string) (int64, error) {
	var count int64
	if err := scanSingleRow(ctx, r.sql, `SELECT COUNT(*) FROM account_proxy_assignments WHERE proxy_group = $1`, []any{groupName}, &count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *proxyRepository) CountAssignmentsByProxy(ctx context.Context) (map[int64]int64, error) {
	return r.countAssignments(ctx, `
		SELECT proxy_id, COUNT(*) FROM account_proxy_assignments GROUP BY proxy_id
	`)
}

func (r *proxyRepository) CountAssignmentsByProxyInGroup(ctx context.Context, groupName string) (map[int64]int64, error) {
	return r.countAssignments(ctx, `
		SELECT proxy_id, COUNT(*) FROM account_proxy_assignments WHERE proxy_group = $1 GROUP BY proxy_id
	`, groupName)
}

func (r *proxyRepository) countAssignments(ctx context.Context, query string, args ...any) (map[int64]int64, error) {
	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	counts := make(map[int64]int64)
	for rows.Next() {
		var proxyID, count int64
		if err := rows.Scan(&proxyID, &count); err != nil {
			return nil, err
		}
		counts[proxyID] = count
	}
	return counts, rows.Err()
}

func (r *proxyRepository) BulkSetAssignments(ctx context.Context, assignments []service.ProxyAssignment) (int, error) {
	if r.sqlDB == nil {
		// Fallback path (no raw SQL client — test stubs only). Not transactional; must NOT be used in production.
		assigned := 0
		for _, a := range assignments {
			if err := r.SetAssignment(ctx, a.AccountID, a.ProxyID, a.ProxyGroup, a.AssignedBy); err != nil {
				return assigned, fmt.Errorf("assign account %d to proxy %d: %w", a.AccountID, a.ProxyID, err)
			}
			assigned++
		}
		return assigned, nil
	}

	tx, err := r.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	assigned := 0
	for _, a := range assignments {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO account_proxy_assignments (account_id, proxy_id, proxy_group, assigned_by)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (account_id) DO UPDATE SET proxy_id = $2, proxy_group = $3, assigned_by = $4, assigned_at = NOW()
		`, a.AccountID, a.ProxyID, a.ProxyGroup, a.AssignedBy)
		if err != nil {
			return 0, fmt.Errorf("assign account %d to proxy %d: %w", a.AccountID, a.ProxyID, err)
		}
		assigned++
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}
	return assigned, nil
}
