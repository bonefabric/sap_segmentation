package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
	offset := 0
out:
	for {
		select {
		case <-s.ctx.Done():
			break out
		default:
			data, err := s.fetchData(offset)
			if err != nil {
				return err
			}
			if len(data) == 0 {
				break
			}
			if err = s.saveData(data); err != nil {
				return err
			}
			offset += s.cfg.ImportBatchSize
			time.Sleep(s.cfg.ConnInterval)
		}
	}
	return nil
}

func (s *ImportService) fetchData(offset int) ([]model.Segmentation, error) {
	client := &http.Client{Timeout: s.cfg.ConnTimeout}
	opts := fmt.Sprintf("%s?p_limit=%d&p_offset=%d", s.cfg.ConnURI, s.cfg.ImportBatchSize, offset)

	slog.Info("fetching data", "uri", opts)

	req, err := http.NewRequestWithContext(s.ctx, "GET", opts, nil)
	if err != nil {
		slog.Error("failed to create http request", "err", err)
		return nil, err
	}
	req.Header.Set("User-Agent", s.cfg.ConnUserAgent)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(s.cfg.ConnAuthLoginPwd)))

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("failed to fetch data", "err", err)
		return nil, err
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
		slog.Error("failed to parse response body", "err", err)
		return nil, err
	}
	return data, nil
}

func (s *ImportService) saveData(data []model.Segmentation) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}
	for _, seg := range data {
		_, err = tx.NamedExec(`
            INSERT INTO segmentation (address_sap_id, adr_segment, segment_id)
            VALUES (:address_sap_id, :adr_segment, :segment_id)
            ON CONFLICT (address_sap_id) DO UPDATE
            SET adr_segment = EXCLUDED.adr_segment, segment_id = EXCLUDED.segment_id`, seg)
		if err != nil {
			if err := tx.Rollback(); err != nil {
				slog.Error("failed to rollback transaction", "err", err)
			}
			return err
		}
	}
	return tx.Commit()
}
