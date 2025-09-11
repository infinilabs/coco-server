package salesforce

import (
	"fmt"
	"strings"
)

// SalesforceSoqlBuilder is a fluent builder for constructing SOQL queries
type SalesforceSoqlBuilder struct {
	tableName string
	fields    []string
	where     string
	orderBy   string
	limit     string
}

// NewSalesforceSoqlBuilder creates a new SOQL query builder
func NewSalesforceSoqlBuilder(tableName string) *SalesforceSoqlBuilder {
	return &SalesforceSoqlBuilder{
		tableName: tableName,
		fields:    make([]string, 0),
	}
}

// WithId adds the Id field to the query
func (b *SalesforceSoqlBuilder) WithId() *SalesforceSoqlBuilder {
	b.fields = append(b.fields, "Id")
	return b
}

// WithDefaultMetafields adds CreatedDate and LastModifiedDate fields
func (b *SalesforceSoqlBuilder) WithDefaultMetafields() *SalesforceSoqlBuilder {
	b.fields = append(b.fields, "CreatedDate", "LastModifiedDate")
	return b
}

// WithFields adds multiple fields to the query
func (b *SalesforceSoqlBuilder) WithFields(fields []string) *SalesforceSoqlBuilder {
	b.fields = append(b.fields, fields...)
	return b
}

// WithField adds a single field to the query
func (b *SalesforceSoqlBuilder) WithField(field string) *SalesforceSoqlBuilder {
	b.fields = append(b.fields, field)
	return b
}

// WithWhere adds a WHERE clause to the query
func (b *SalesforceSoqlBuilder) WithWhere(whereString string) *SalesforceSoqlBuilder {
	if whereString != "" {
		b.where = fmt.Sprintf("WHERE %s", whereString)
	}
	return b
}

// WithOrderBy adds an ORDER BY clause to the query
func (b *SalesforceSoqlBuilder) WithOrderBy(orderByString string) *SalesforceSoqlBuilder {
	if orderByString != "" {
		b.orderBy = fmt.Sprintf("ORDER BY %s", orderByString)
	}
	return b
}

// WithLimit adds a LIMIT clause to the query
func (b *SalesforceSoqlBuilder) WithLimit(limit int) *SalesforceSoqlBuilder {
	if limit > 0 {
		b.limit = fmt.Sprintf("LIMIT %d", limit)
	}
	return b
}

// WithJoin adds a subquery join to the query
func (b *SalesforceSoqlBuilder) WithJoin(join string) *SalesforceSoqlBuilder {
	if join != "" {
		b.fields = append(b.fields, fmt.Sprintf("(\n%s)\n", join))
	}
	return b
}

// Build constructs the final SOQL query string
func (b *SalesforceSoqlBuilder) Build() string {
	if len(b.fields) == 0 {
		return ""
	}

	// Remove duplicate fields while preserving order
	fieldMap := make(map[string]bool)
	var uniqueFields []string
	for _, field := range b.fields {
		if !fieldMap[field] {
			fieldMap[field] = true
			uniqueFields = append(uniqueFields, field)
		}
	}

	// Build the query parts
	queryParts := []string{
		fmt.Sprintf("SELECT %s", strings.Join(uniqueFields, ", ")),
		fmt.Sprintf("FROM %s", b.tableName),
	}

	// Add optional clauses
	if b.where != "" {
		queryParts = append(queryParts, b.where)
	}
	if b.orderBy != "" {
		queryParts = append(queryParts, b.orderBy)
	}
	if b.limit != "" {
		queryParts = append(queryParts, b.limit)
	}

	return strings.Join(queryParts, "\n")
}

// GetFields returns the current fields list
func (b *SalesforceSoqlBuilder) GetFields() []string {
	return b.fields
}

// GetTableName returns the table name
func (b *SalesforceSoqlBuilder) GetTableName() string {
	return b.tableName
}

// Clone creates a copy of the builder
func (b *SalesforceSoqlBuilder) Clone() *SalesforceSoqlBuilder {
	clone := &SalesforceSoqlBuilder{
		tableName: b.tableName,
		fields:    make([]string, len(b.fields)),
		where:     b.where,
		orderBy:   b.orderBy,
		limit:     b.limit,
	}
	copy(clone.fields, b.fields)
	return clone
}
