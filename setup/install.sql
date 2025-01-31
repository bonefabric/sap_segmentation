CREATE TABLE segmentation
(
    id             SERIAL PRIMARY KEY,
    address_sap_id VARCHAR(255) UNIQUE NOT NULL,
    adr_segment    VARCHAR(16)         NOT NULL,
    segment_id     BIGINT              NOT NULL
);