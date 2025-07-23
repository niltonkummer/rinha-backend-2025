package db

// change to use postgres with pgx package
// Package db provides an adapter for interacting with the SQLite database for stats management.

import (
	"context"
	"database/sql"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/shopspring/decimal"
	"log/slog"
	"niltonkummer/rinha-2025/pkg/adapters"
	"niltonkummer/rinha-2025/pkg/models"
	"strconv"
	"sync"
	"time"
)

var DB *sql.DB

type stats struct {
	Amount        decimal.Decimal `json:"amount"`         // Amount processed in the stats entry
	Fallback      bool            `json:"fallback"`       // Indicates if this is a fallback entry
	ProcessedAt   time.Time       `json:"processed_at"`   // Timestamp when the stats were processed
	CorrelationID string          `json:"correlation_id"` // Correlation ID for tracking the request
}

type sqlStatsAdapter struct {
	db              *sql.DB
	log             *slog.Logger // Logger for logging messages
	lazyInsertStats []stats
	mutex           sync.Mutex // Mutex to protect access to lazyInsertStats
	lastInsertTime  time.Time  // Last time stats were inserted
}

func NewStatsAdapter(connString string, log *slog.Logger) (adapters.StatsAdapter, error) {
	ctx := context.TODO()

	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		log.Error("Failed to connect to the database", "error", err) // Log an error if the connection fails
		return nil, err                                              // Return nil and the error to the caller
	}

	DB = stdlib.OpenDBFromPool(pool)

	sqlStmt := `
 CREATE TABLE IF NOT EXISTS stats (
		correlation_id UUID PRIMARY KEY, -- Use UUID for unique correlation IDs
		amount DECIMAL(10, 2) NOT NULL DEFAULT 0.00, -- Use DECIMAL for precise monetary values
     	fallback BOOLEAN NOT NULL DEFAULT FALSE, -- Use BOOLEAN for true/false values
		processed_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
 );`
	// SQL statement to create the todos table if it doesn't exist

	_, err = DB.ExecContext(ctx, sqlStmt)
	if err != nil {
		log.Error("Error creating table", "err", err) // Log an error if the table creation fails
		return nil, err
	}

	sa := &sqlStatsAdapter{
		log: log,
		db:  DB, // Assign the database connection to the adapter
	}

	/*go func() {
		for ; ; time.Sleep(time.Second) {

			sa.mutex.Lock()
			if len(sa.lazyInsertStats) == 0 { // If there are no stats to insert, skip the bulk insertion
				sa.mutex.Unlock()
				continue
			}

			err := sa.insertBulkStats() // Attempt to insert the accumulated stats into the database
			if err != nil {
				sa.log.Error("Failed to insert bulk stats", "error", err) // Log an error if the bulk insertion fails
			}
			sa.lazyInsertStats = nil // Clear the slice after insertion
			sa.mutex.Unlock()
		}
	}()*/

	return sa, nil // Return the statsAdapter instance
}

func (s *sqlStatsAdapter) InsertStats(correlationID string, amount decimal.Decimal, fallback bool, requestedAt time.Time) error {

	tx, err := s.db.BeginTx(context.TODO(), &sql.TxOptions{Isolation: sql.LevelSerializable}) // Start a new transaction with SERIALIZABLE isolation level
	if err != nil {
		s.log.Error("Failed to begin transaction", "error", err) // Log an error if the transaction can't be started
		return err                                               // Return the error to the caller
	}
	query := "INSERT INTO stats (processed_at, correlation_id, amount, fallback) VALUES ($1, $2, $3, $4) ON CONFLICT (correlation_id) DO NOTHING"
	ctx := context.TODO()
	_, err = tx.ExecContext(ctx, query, time.Now(), correlationID, amount, fallback)
	if err != nil {
		tx.Rollback()
		s.log.Error("Failed to insert stats", "error", err) // Log an error if the insertion fails
		return err                                          // Return the error to the caller
	}
	return tx.Commit()

}

func (s *sqlStatsAdapter) InsertStats2(correlationID string, amount decimal.Decimal, isFallback bool) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	// Update the last insertion time to the current time
	//append logic to insert stats into the database
	s.lazyInsertStats = append(s.lazyInsertStats, stats{
		Amount:        amount,
		Fallback:      isFallback,
		ProcessedAt:   time.Now(),
		CorrelationID: correlationID,
	})
	// If the lazyInsertStats slice has reached a certain size, insert them into the database
	if len(s.lazyInsertStats) >= 500 { // Adjust the threshold as needed
		err := s.insertBulkStats()
		if err != nil {
			s.log.Error("Failed to insert bulk stats", "error", err) // Log an error if the bulk insertion fails
			return err                                               // Return the error to the caller
		}
		s.lastInsertTime = time.Now()
		s.lazyInsertStats = nil // Clear the slice after insertion
	}
	/*_, err := s.db.ExecContext(context.TODO(), "INSERT INTO stats (processed_at, correlation_id, amount, fallback) VALUES ($1, $2, $3, $4)", time.Now(), correlationID, amount, isFallback)
	if err != nil {
		return err // Return an error if the statement preparation fails
	}*/

	return nil // Return nil if the insertion is successful
}

// accumulates a bulk of stats and inserts them into the database
func (s *sqlStatsAdapter) insertBulkStats() error {
	ctx := context.TODO()

	argsNum := 1
	query := "INSERT INTO stats (processed_at, correlation_id, amount, fallback) VALUES "
	params := make([]interface{}, 0, len(s.lazyInsertStats)*4) // Preallocate the slice for parameters
	for i, stat := range s.lazyInsertStats {
		if i > 0 {
			query += ", "
		}
		query += "($" + strconv.Itoa(argsNum) + ", $" + strconv.Itoa(argsNum+1) + ", $" + strconv.Itoa(argsNum+2) + ", $" + strconv.Itoa(argsNum+3) + ")"
		argsNum += 4 // Increment the argument number for the next set of parameters
		params = append(params, stat.ProcessedAt, stat.CorrelationID, stat.Amount, stat.Fallback)
	}
	s.log.Info("Executing bulk insert", "query", query, "params", params)
	// Prepare the statement with the correct number of placeholders
	_, err := s.db.ExecContext(ctx, query, params...)
	if err != nil {
		return err // Return an error if the statement preparation fails
	}
	params = params[:0]
	return nil
}

func (s *sqlStatsAdapter) RetrieveTotalStatsByPeriod(fallback bool, start, end time.Time) (*models.PaymentsSummaryResponse, error) {
	// TODO: Implement the logic to retrieve total stats by period
	panic("not implemented yet") // This is a placeholder panic to indicate that the method is not yet implemented
	/*row := s.db.QueryRowContext(context.TODO(), "SELECT COUNT(1), SUM(amount) FROM stats WHERE processed_at BETWEEN $1 AND $2 AND fallback = $3", start, end, fallback)

	var count int
	var amount decimal.Decimal
	// Execute the statement with the start and end times and scan the result into count
	err := row.Scan(&count, &amount)
	if err != nil {
		return nil, err // Return an error if the query fails
	}

	// Return the count as a PaymentsSummary struct
	return &models.PaymentsSummary{
		TotalRequests: count,
		TotalAmount:   amount, // Assuming TotalAmount is the count for simplicity
	}, nil*/
}
