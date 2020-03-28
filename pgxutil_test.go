package pgxutil_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgxutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withTx(t testing.TB, f func(ctx context.Context, tx pgx.Tx)) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	conn := connectPG(t, ctx)
	defer closeConn(t, conn)

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	f(ctx, tx)
}

func connectPG(t testing.TB, ctx context.Context) *pgx.Conn {
	config, err := pgx.ParseConfig(fmt.Sprintf("database=%s", os.Getenv("TEST_DATABASE")))
	require.NoError(t, err)
	config.OnNotice = func(_ *pgconn.PgConn, n *pgconn.Notice) {
		t.Logf("PostgreSQL %s: %s", n.Severity, n.Message)
	}

	conn, err := pgx.ConnectConfig(ctx, config)
	require.NoError(t, err)
	return conn
}

func closeConn(t testing.TB, conn *pgx.Conn) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, conn.Close(ctx))
}

func TestSelectOneCommonErrors(t *testing.T) {
	t.Parallel()
	withTx(t, func(ctx context.Context, tx pgx.Tx) {
		tests := []struct {
			sql    string
			err    string
			result interface{}
		}{
			{"select 42::float8 where 1=0", "no rows in result set", nil},
			{"select 42::float8 from generate_series(1,2)", "multiple rows in result set", nil},
			{"select", "no columns in result set", nil},
			{"select 1, 2", "multiple columns in result set", nil},
		}
		for i, tt := range tests {
			v, err := pgxutil.SelectValue(ctx, tx, tt.sql)
			if tt.err == "" {
				assert.NoErrorf(t, err, "%d. %s", i, tt.sql)
			} else {
				assert.EqualErrorf(t, err, tt.err, "%d. %s", i, tt.sql)
			}
			assert.Equalf(t, tt.result, v, "%d. %s", i, tt.sql)
		}
	})
}

func TestSelectString(t *testing.T) {
	t.Parallel()
	withTx(t, func(ctx context.Context, tx pgx.Tx) {
		tests := []struct {
			sql    string
			result string
		}{
			{"select 'Hello, world!'", "Hello, world!"},
			{"select 42", "42"},
		}
		for i, tt := range tests {
			v, err := pgxutil.SelectString(ctx, tx, tt.sql)
			assert.NoErrorf(t, err, "%d. %s", i, tt.sql)
			assert.Equalf(t, tt.result, v, "%d. %s", i, tt.sql)
		}
	})
}

func TestSelectStringColumn(t *testing.T) {
	t.Parallel()
	withTx(t, func(ctx context.Context, tx pgx.Tx) {
		tests := []struct {
			sql    string
			result []string
		}{
			{"select format('Hello %s', n) from generate_series(1,2) n", []string{"Hello 1", "Hello 2"}},
			{"select 'Hello, world!' where false", nil},
		}
		for i, tt := range tests {
			v, err := pgxutil.SelectStringColumn(ctx, tx, tt.sql)
			assert.NoErrorf(t, err, "%d. %s", i, tt.sql)
			assert.Equalf(t, tt.result, v, "%d. %s", i, tt.sql)
		}
	})
}

func TestSelectByteSlice(t *testing.T) {
	t.Parallel()
	withTx(t, func(ctx context.Context, tx pgx.Tx) {
		tests := []struct {
			sql    string
			result []byte
		}{
			{"select 'Hello, world!'", []byte("Hello, world!")},
			{"select 42", []byte{0, 0, 0, 42}},
		}
		for i, tt := range tests {
			v, err := pgxutil.SelectByteSlice(ctx, tx, tt.sql)
			assert.NoErrorf(t, err, "%d. %s", i, tt.sql)
			assert.Equalf(t, tt.result, v, "%d. %s", i, tt.sql)
		}
	})
}

func TestSelectByteSliceColumn(t *testing.T) {
	t.Parallel()
	withTx(t, func(ctx context.Context, tx pgx.Tx) {
		tests := []struct {
			sql    string
			result [][]byte
		}{
			{"select format('Hello %s', n) from generate_series(1,2) n", [][]byte{[]byte("Hello 1"), []byte("Hello 2")}},
			{"select 'Hello, world!' where false", nil},
		}
		for i, tt := range tests {
			v, err := pgxutil.SelectByteSliceColumn(ctx, tx, tt.sql)
			assert.NoErrorf(t, err, "%d. %s", i, tt.sql)
			assert.Equalf(t, tt.result, v, "%d. %s", i, tt.sql)
		}
	})
}

func TestSelectInt64(t *testing.T) {
	t.Parallel()
	withTx(t, func(ctx context.Context, tx pgx.Tx) {
		tests := []struct {
			sql    string
			result int64
		}{
			{"select 99999999999::bigint", 99999999999},
			{"select 42::smallint", 42},
		}
		for i, tt := range tests {
			v, err := pgxutil.SelectInt64(ctx, tx, tt.sql)
			assert.NoErrorf(t, err, "%d. %s", i, tt.sql)
			assert.Equalf(t, tt.result, v, "%d. %s", i, tt.sql)
		}
	})
}

func TestSelectInt64Column(t *testing.T) {
	t.Parallel()
	withTx(t, func(ctx context.Context, tx pgx.Tx) {
		tests := []struct {
			sql    string
			result []int64
		}{
			{"select generate_series(1,2)", []int64{1, 2}},
			{"select 42 where false", nil},
		}
		for i, tt := range tests {
			v, err := pgxutil.SelectInt64Column(ctx, tx, tt.sql)
			assert.NoErrorf(t, err, "%d. %s", i, tt.sql)
			assert.Equalf(t, tt.result, v, "%d. %s", i, tt.sql)
		}
	})
}

func TestSelectFloat64(t *testing.T) {
	t.Parallel()
	withTx(t, func(ctx context.Context, tx pgx.Tx) {
		tests := []struct {
			sql    string
			result float64
		}{
			{"select 1.2345::float8", 1.2345},
			{"select 1.23::float4", 1.23},
			{"select 1.2345::numeric", 1.2345},
			{"select 99999999999::bigint", 99999999999},
			{"select 42::smallint", 42},
		}
		for i, tt := range tests {
			v, err := pgxutil.SelectFloat64(ctx, tx, tt.sql)
			assert.NoErrorf(t, err, "%d. %s", i, tt.sql)
			assert.Equalf(t, tt.result, v, "%d. %s", i, tt.sql)
		}
	})
}

func TestSelectFloat64Column(t *testing.T) {
	t.Parallel()
	withTx(t, func(ctx context.Context, tx pgx.Tx) {
		tests := []struct {
			sql    string
			result []float64
		}{
			{"select n + 0.5 from generate_series(1,2) n", []float64{1.5, 2.5}},
			{"select 42.0 where false", nil},
		}
		for i, tt := range tests {
			v, err := pgxutil.SelectFloat64Column(ctx, tx, tt.sql)
			assert.NoErrorf(t, err, "%d. %s", i, tt.sql)
			assert.Equalf(t, tt.result, v, "%d. %s", i, tt.sql)
		}
	})
}

func TestSelectDecimal(t *testing.T) {
	t.Parallel()
	withTx(t, func(ctx context.Context, tx pgx.Tx) {
		tests := []struct {
			sql    string
			result string
		}{
			{"select 1.2345::numeric", "1.2345"},
			{"select 1.2345::float8", "1.2345"},
			{"select 1.23::float4", "1.23"},
			{"select 99999999999::bigint", "99999999999"},
			{"select 42::smallint", "42"},
		}
		for i, tt := range tests {
			v, err := pgxutil.SelectDecimal(ctx, tx, tt.sql)
			assert.NoErrorf(t, err, "%d. %s", i, tt.sql)
			assert.Equalf(t, tt.result, v.String(), "%d. %s", i, tt.sql)
		}
	})
}

func TestSelectDecimalColumn(t *testing.T) {
	t.Parallel()
	withTx(t, func(ctx context.Context, tx pgx.Tx) {
		tests := []struct {
			sql    string
			result []string
		}{
			{"select n + 0.5 from generate_series(1,2) n", []string{"1.5", "2.5"}},
			{"select 42.0 where false", nil},
		}
		for i, tt := range tests {
			v, err := pgxutil.SelectDecimalColumn(ctx, tx, tt.sql)
			assert.NoErrorf(t, err, "%d. %s", i, tt.sql)
			if assert.Equalf(t, len(tt.result), len(v), "%d. %s", i, tt.sql) {
				for j := range v {
					assert.Equalf(t, tt.result[j], v[j].String(), "%d. %s - %d", i, tt.sql, j)
				}
			}
		}
	})
}

func TestSelectUUID(t *testing.T) {
	t.Parallel()
	withTx(t, func(ctx context.Context, tx pgx.Tx) {
		tests := []struct {
			sql    string
			result uuid.UUID
		}{
			{"select '27fd10c1-bccc-4efd-9fea-093f86c95089'::uuid", uuid.FromStringOrNil("27fd10c1-bccc-4efd-9fea-093f86c95089")},
		}
		for i, tt := range tests {
			v, err := pgxutil.SelectUUID(ctx, tx, tt.sql)
			assert.NoErrorf(t, err, "%d. %s", i, tt.sql)
			assert.Equalf(t, tt.result, v, "%d. %s", i, tt.sql)
		}
	})
}

func TestSelectUUIDColumn(t *testing.T) {
	t.Parallel()
	withTx(t, func(ctx context.Context, tx pgx.Tx) {
		tests := []struct {
			sql    string
			result []uuid.UUID
		}{
			{
				sql: "select format('27fd10c1-bccc-4efd-9fea-093f86c9508%s', n)::uuid from generate_series(1,2) n",
				result: []uuid.UUID{
					uuid.FromStringOrNil("27fd10c1-bccc-4efd-9fea-093f86c95081"),
					uuid.FromStringOrNil("27fd10c1-bccc-4efd-9fea-093f86c95082"),
				},
			},
			{
				sql:    "select '27fd10c1-bccc-4efd-9fea-093f86c95089'::uuid where false",
				result: nil,
			},
		}
		for i, tt := range tests {
			v, err := pgxutil.SelectUUIDColumn(ctx, tx, tt.sql)
			assert.NoErrorf(t, err, "%d. %s", i, tt.sql)
			assert.Equalf(t, tt.result, v, "%d. %s", i, tt.sql)
		}
	})
}

func TestSelectValue(t *testing.T) {
	t.Parallel()
	withTx(t, func(ctx context.Context, tx pgx.Tx) {
		tests := []struct {
			sql    string
			result interface{}
		}{
			{"select 'Hello'", "Hello"},
			{"select 42", int32(42)},
			{"select 1.23::float4", float32(1.23)},
			{"select null::float4", nil},
		}
		for i, tt := range tests {
			v, err := pgxutil.SelectValue(ctx, tx, tt.sql)
			assert.NoErrorf(t, err, "%d. %s", i, tt.sql)
			assert.Equalf(t, tt.result, v, "%d. %s", i, tt.sql)
		}
	})
}

func TestSelectValueColumn(t *testing.T) {
	t.Parallel()
	withTx(t, func(ctx context.Context, tx pgx.Tx) {
		tests := []struct {
			sql    string
			result []interface{}
		}{
			{"select n from generate_series(1,2) n", []interface{}{int32(1), int32(2)}},
		}
		for i, tt := range tests {
			v, err := pgxutil.SelectValueColumn(ctx, tx, tt.sql)
			assert.NoErrorf(t, err, "%d. %s", i, tt.sql)
			assert.Equalf(t, tt.result, v, "%d. %s", i, tt.sql)
		}
	})
}

func TestSelectMap(t *testing.T) {
	t.Parallel()
	withTx(t, func(ctx context.Context, tx pgx.Tx) {
		tests := []struct {
			sql    string
			result map[string]interface{}
		}{
			{"select 'Adam' as name, 72 as height", map[string]interface{}{"name": "Adam", "height": int32(72)}},
		}
		for i, tt := range tests {
			v, err := pgxutil.SelectMap(ctx, tx, tt.sql)
			assert.NoErrorf(t, err, "%d. %s", i, tt.sql)
			assert.Equalf(t, tt.result, v, "%d. %s", i, tt.sql)
		}
	})
}

func TestSelectMapColumn(t *testing.T) {
	t.Parallel()
	withTx(t, func(ctx context.Context, tx pgx.Tx) {
		tests := []struct {
			sql    string
			result []map[string]interface{}
		}{
			{
				sql: "select n as a, n+1 as b from generate_series(1,2) n",
				result: []map[string]interface{}{
					{"a": int32(1), "b": int32(2)},
					{"a": int32(2), "b": int32(3)},
				},
			},
		}
		for i, tt := range tests {
			v, err := pgxutil.SelectMapColumn(ctx, tx, tt.sql)
			assert.NoErrorf(t, err, "%d. %s", i, tt.sql)
			assert.Equalf(t, tt.result, v, "%d. %s", i, tt.sql)
		}
	})
}

func TestSelectStringMap(t *testing.T) {
	t.Parallel()
	withTx(t, func(ctx context.Context, tx pgx.Tx) {
		tests := []struct {
			sql    string
			result map[string]string
		}{
			{"select 'Adam' as name, 72 as height", map[string]string{"name": "Adam", "height": "72"}},
		}
		for i, tt := range tests {
			v, err := pgxutil.SelectStringMap(ctx, tx, tt.sql)
			assert.NoErrorf(t, err, "%d. %s", i, tt.sql)
			assert.Equalf(t, tt.result, v, "%d. %s", i, tt.sql)
		}
	})
}

func TestSelectStringMapColumn(t *testing.T) {
	t.Parallel()
	withTx(t, func(ctx context.Context, tx pgx.Tx) {
		tests := []struct {
			sql    string
			result []map[string]string
		}{
			{
				sql: "select n as a, n+1 as b from generate_series(1,2) n",
				result: []map[string]string{
					{"a": "1", "b": "2"},
					{"a": "2", "b": "3"},
				},
			},
		}
		for i, tt := range tests {
			v, err := pgxutil.SelectStringMapColumn(ctx, tx, tt.sql)
			assert.NoErrorf(t, err, "%d. %s", i, tt.sql)
			assert.Equalf(t, tt.result, v, "%d. %s", i, tt.sql)
		}
	})
}
