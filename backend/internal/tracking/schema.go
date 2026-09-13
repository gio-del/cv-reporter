package tracking

// Schema versions of the persisted Job Listing and Application records
// (issue #100, ADR-0034). Each record carries its own schemaVersion key —
// in a Job Listing's frontmatter, at the top level of an Application — and
// each GenerationRecord inside an Application carries one too, because an
// Application written long ago keeps accumulating Generations afterwards.
//
// The version says what an absent field means:
//
//   - On a record at CurrentSchemaVersion, absent means genuinely none: a
//     Generation with no sourceSnippetIds used no Cover Letter Snippet, one
//     with no entryIds recorded no Selection, a Job Listing always carries
//     freshnessStatus.
//   - On a LegacySchemaVersion record, absent means unknowable: the record
//     may simply predate the field.
//
// Adding the next version is: bump CurrentSchemaVersion, stamp the new
// field's guarantee in this comment, teach MigrateRecords the step from the
// previous version (what it can backfill and what it must report as
// unknowable), and record the change in ADR-0034. Never add a new "absent
// for records written before this field existed" tolerance to a reader
// instead.
const (
	// LegacySchemaVersion is every record written before schemaVersion
	// existed. It is the zero value, so a file with no schemaVersion key
	// reads as legacy and renders without the key.
	LegacySchemaVersion = 0

	// CurrentSchemaVersion is what Save stamps on new Job Listings and
	// Applications, what RecordGeneration stamps on new Generations, and
	// what MigrateRecords brings legacy Job Listings and Applications to.
	//
	// Version 1 guarantees: a Job Listing carries freshnessStatus; an
	// Application created at v1 carries statusUpdatedAt and a statusHistory
	// starting at Saved (a migrated one carries them only if it was still
	// at Saved when migrated — see MigrateRecords); archived and notes are
	// optional keys whose absence means false and no Notes respectively.
	CurrentSchemaVersion = 1
)

// IsLegacy reports whether g was recorded before GenerationRecords carried
// a schema version, in which case an absent SourceSnippetIDs, EntryIDs,
// Usage or Language is unknowable rather than genuinely empty.
func (g GenerationRecord) IsLegacy() bool {
	return g.SchemaVersion == LegacySchemaVersion
}
