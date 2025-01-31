package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"golang.org/x/sync/errgroup"
	"io"
	"log/slog"
	"net/http"
	"sap_segmentation/internal/config"
	"sap_segmentation/internal/model"
	"time"

	"github.com/jmoiron/sqlx"
)

type ImportService struct {
	db  *sqlx.DB
	cfg *config.Config
	ctx context.Context
}

func NewImportService(db *sqlx.DB, cfg *config.Config, ctx context.Context) *ImportService {
	return &ImportService{db: db, cfg: cfg, ctx: ctx}
}

func (s *ImportService) ImportData() error {
	data := make(chan model.Segmentation, 10*s.cfg.ImportBatchSize)

	eg := &errgroup.Group{}

	eg.Go(func() error {
		return s.saveData(data)
	})

	eg.Go(func() error {
		return s.loadData(data)
	})

	return eg.Wait()
}

func (s *ImportService) loadData(data chan<- model.Segmentation) error {
	defer close(data)
	offset := 0
	for {
		p, err := s.fetchData(offset)

		if err != nil || len(p) == 0 {
			return err
		}
		for _, seg := range p {
			data <- seg
		}

		offset += s.cfg.ImportBatchSize
		time.Sleep(s.cfg.ConnInterval)
	}
}

func (s *ImportService) fetchData(offset int) ([]model.Segmentation, error) {
	client := &http.Client{Timeout: s.cfg.ConnTimeout}
	opts := fmt.Sprintf("%s?p_limit=%d&p_offset=%d", s.cfg.ConnURI, s.cfg.ImportBatchSize, offset)

	req, err := http.NewRequestWithContext(s.ctx, "GET", opts, nil)
	if err != nil {
		slog.Error("failed to create http request", "err", err)
		return nil, err
	}
	req.Header.Set("User-Agent", s.cfg.ConnUserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(s.cfg.ConnAuthLoginPwd)))

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("failed to fetch data", "err", err)
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch data, status code: %d", resp.StatusCode)
	}

	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			slog.Error("failed to close response body", "err", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("failed to read response body", "err", err)
		return nil, err
	}

	var data []model.Segmentation
	if err = json.Unmarshal(body, &data); err != nil {
		slog.Error("failed to parse response body", "err", err, "body", string(body))
		return nil, err
	}
	slog.Info("fetched data", "uri", opts)

	return data, nil
}

func (s *ImportService) saveData(data <-chan model.Segmentation) error {
	for seg := range data {
		tx, err := s.db.Beginx()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		_, err = tx.NamedExec(`
            INSERT INTO segmentation (address_sap_id, adr_segment, segment_id)
            VALUES (:address_sap_id, :adr_segment, :segment_id)
            ON CONFLICT (address_sap_id) DO UPDATE
            SET adr_segment = EXCLUDED.adr_segment, segment_id = EXCLUDED.segment_id`, seg)
		if err != nil {
			if err := tx.Rollback(); err != nil {
				slog.Error("failed to rollback transaction", "err", err)
			}
			return fmt.Errorf("failed to update segmentation: %w", err)
		}
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}
	return nil
}
