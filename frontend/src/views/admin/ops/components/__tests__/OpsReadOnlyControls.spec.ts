import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentsDir = resolve(dirname(fileURLToPath(import.meta.url)), '..')

describe('ops read-only controls', () => {
  it('guards alert silence and manual resolution actions', () => {
    const source = readFileSync(resolve(componentsDir, 'OpsAlertEventsCard.vue'), 'utf8')
    expect(source).toContain('if (props.readOnly) return')
    expect(source).toContain('v-if="!props.readOnly"')
  })

  it('guards runtime settings and cleanup actions', () => {
    const source = readFileSync(resolve(componentsDir, 'OpsSystemLogTable.vue'), 'utf8')
    expect(source.match(/if \(props\.readOnly\) return/g)).toHaveLength(3)
    expect(source).toContain('v-if="!props.readOnly"')
  })
})
