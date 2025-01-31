package model

type Segmentation struct {
	ID           int64  `db:"id"`
	AddressSapID string `db:"address_sap_id"`
	AdrSegment   string `db:"adr_segment"`
	SegmentID    int64  `db:"segment_id"`
}
