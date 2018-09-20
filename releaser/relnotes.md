# Version 2: Moves team secrets to dedicated spaces

- Updates team secrets to now use dedicates spaces instead of providing secrets to each team member individually. This allows secrets to persist if all members of a team rotate out.
- Updates Vault policy generation to allow for generating access to team spaces.
- Allows for case-insensitive lookups without breaking backwards compatibility for previously created secrets.
