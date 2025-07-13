package unions

// Discriminator interface allows types to specify their discriminator value.
// When a type implements this interface, the union unmarshal logic can use
// the discriminator value for efficient type selection instead of trying each type.
type Discriminator interface {
	// DiscriminatorValue returns the unique discriminator value for this type.
	// This value should match what's in the JSON discriminator field.
	DiscriminatorValue() string
}

// DiscriminatorField interface allows union types to specify which field
// contains the discriminator value. This is optional - if not implemented,
// the system will look for common discriminator field names like "type" or "kind".
type DiscriminatorField interface {
	// DiscriminatorFieldName returns the name of the JSON field that contains
	// the discriminator value (e.g., "type", "kind", "@type").
	DiscriminatorFieldName() string
}