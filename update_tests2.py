import os

path = "server/internal/storage/db/users_test.go"
content = open(path, "r").read()

# 1. Add language to fullUserCols AddRow calls that end with "", "")
# Full cols AddRow lines end with: true, true, "")
content = content.replace(
    'true, true, "")',
    'true, true, "", "zh-Hans")'
)

# 2. Add language to all SELECT * FROM users regex patterns
# Replace "headline FROM users WHERE" with "headline, language FROM users WHERE"
content = content.replace(
    "headline FROM users WHERE",
    "headline, language FROM users WHERE"
)

# 3. Add zh-Hans to AddRow for full cols with password
# Pattern end: "", 0, "", true, true, "")
# Already handled above

# 4. Handle GetByIDs and other non-full-col AddRow patterns
# userColumnOrder AddRow lines end with: true, false, "") etc.
content = content.replace(
    'true, false, "")',
    'true, false, "", "zh-Hans")'
)
content = content.replace(
    'false, true, "")',
    'false, true, "", "zh-Hans")'
)
content = content.replace(
    'false, false, "")',
    'false, false, "", "zh-Hans")'
)

# 5. Agent AddRow patterns (userColumnOrder)
# Agent pattern: 0, false, "")
content = content.replace(
    '0, false, "")',
    '0, false, "", "zh-Hans")'
)
content = content.replace(
    '0, true, "")',
    '0, true, "", "zh-Hans")'
)

print("Done")
