## Summary

Describe the change and why it is needed.

## Type of change

- [ ] Bug fix
- [ ] New feature
- [ ] D-Bus interface change (XML updated, both sides updated)
- [ ] i18n (new strings added, `.ts` files updated)
- [ ] Documentation / packaging

## Testing

- [ ] `cd daemon && go build ./cmd` passes
- [ ] `cmake --build build` passes
- [ ] Tested against a real WhatsApp account (if applicable)

## Checklist

- [ ] No hardcoded colors (use `Kirigami.Theme.*`)
- [ ] All user-visible strings wrapped in `qsTr()`
- [ ] `Accessible.*` properties added for new interactive elements
- [ ] D-Bus XML updated if interface changed
