CREATE TABLE fields (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    field_type VARCHAR(20) NOT NULL CHECK (field_type IN ('text', 'number', 'select', 'boolean')),
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('user', 'team')),
    required BOOLEAN DEFAULT FALSE,
    options JSONB,
    order_index INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE field_values (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    field_id uuid NOT NULL REFERENCES fields(id) ON DELETE CASCADE,
    entity_id uuid NOT NULL,
    value TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (field_id, entity_id)
);

CREATE INDEX idx_field_values_entity ON field_values (entity_id);
CREATE INDEX idx_fields_entity_type ON fields (entity_type);
