# Image Definition Schema

## Schema Fields

TBD

## Versioning

There are three types of changes that may be made to the definition schema between releases:
* New optional additions
* New required additions that would break existing definitions
* Changes to existing fields (renaming, changing types, removing a field)

The definition schema versioning will adhere to the following rules:

**1. Each time a new release comes out that requires a schema change (regardless of which type of change occurred),
the schema version will be bumped to match the EIB version.**

For example, if the latest version of the schema is 1.1, and the schema is not changed until the 1.4 release, the next
(or current, at the time) schema version will be 1.4.

**2. Within a major release (i.e. 2.x, 3.x), all minor releases will be backward compatible with all schema versions
since the x.0 release in that stream.**

**3. Breaking changes, such as new required fields or changes to existing fields, may only be done at the start of
a new major release (x.0)**

For example, assume the following releases have occurred. For clarity, the schema versions will be letters in this
example. The following describes the schema version compatibility:

| EIB Version | Supported Schema Versions | Notes                                                                                                                           |
|-------------|---------------------------|---------------------------------------------------------------------------------------------------------------------------------|
| 1.0         | a                         | When EIB 1.0 is released, the schema version is 'a'                                                                             |
| 1.1         | a                         | EIB 1.1 does not make any changes to the schema                                                                                 |
| 1.2         | a, b                      | EIB 1.2 introduces non-breaking schema changes, making the current version 'b'. Schema version 'a' is still supported.          |
| 1.3         | a, b, c                   | EIB 1.3 also introduces non-breaking schema changes, bumping the version to 'c' while supporting all versions in the 1.x stream |
| 2.0         | d                         | At the next major EIB release, a new schema version number is used, dropping backward compatibility with the 1.x versions       |
| 2.1         | d, e                      | The pattern continues throughout the 2.x stream                                                                                 |

Keeping in mind the first rule, the values for the letters in the above table are as follows:

| Key | Schema Version |
|-----|----------------|
| a   | 1.0            |
| b   | 1.2            | 
| c   | 1.3            |
| d   | 2.0            |
| e   | 2.1            |

For each new schema version, EIB will provide a migration path to ease the transition (likely documentation, however 
where applicable a small script may be provided if possible).