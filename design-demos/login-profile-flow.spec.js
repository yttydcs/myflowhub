const { test, expect } = require('playwright/test')
const path = require('node:path')
const { pathToFileURL } = require('node:url')

const prototypeURL = pathToFileURL(path.join(__dirname, 'login-profile-flow.html')).href

test.use({ channel: 'chrome' })

test('existing profiles expose registered and pending connection states', async ({ page }) => {
  const errors = []
  page.on('pageerror', (error) => errors.push(error.message))

  await page.setViewportSize({ width: 1440, height: 900 })
  await page.goto(prototypeURL)

  await expect(page.getByRole('tab', { name: '使用现有 Profile' })).toHaveAttribute('aria-selected', 'true')
  await expect(page.getByRole('button', { name: /^个人工作区 Node/ })).toBeVisible()
  await expect(page.getByRole('button', { name: /连接父节点/ })).toBeVisible()
  await page.screenshot({ path: path.join(__dirname, 'login-profile-existing.png') })

  await page.getByRole('button', { name: /^实验室节点 lab-gateway/ }).click()
  await expect(page.getByText('等待审批', { exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: /检查审批并连接/ })).toBeVisible()
  await page.screenshot({ path: path.join(__dirname, 'login-profile-pending.png') })

  expect(errors).toEqual([])
})

test('first connection switches between approval and permit without Node ID input', async ({ page }) => {
  const errors = []
  page.on('pageerror', (error) => errors.push(error.message))

  await page.setViewportSize({ width: 1440, height: 900 })
  await page.goto(prototypeURL)
  await page.getByRole('tab', { name: '首次连接' }).click()

  await expect(page.getByLabel('Profile 名称')).toBeVisible()
  await expect(page.getByLabel('父节点地址')).toBeVisible()
  await expect(page.getByText('Node ID 将在准入成功后由 Authority 下发')).toBeVisible()
  await expect(page.getByLabel('预期父 Node ID')).toBeHidden()
  await expect(page.getByRole('button', { name: /提交申请/ })).toBeVisible()

  await page.getByRole('button', { name: '使用 Permit' }).click()
  await expect(page.getByLabel('本机公钥')).toBeVisible()
  await expect(page.getByLabel('Enrollment Permit')).toBeVisible()
  await expect(page.getByText(/我确认首次连接时信任/)).toBeHidden()
  await page.getByRole('button', { name: '生成并复制' }).click()
  await expect(page.getByLabel('本机公钥')).not.toHaveValue('')
  await page.getByLabel('Enrollment Permit').fill('{"version":1,"permit_id":"demo"}')
  await page.locator('.panel-title').click()
  await page.waitForTimeout(2300)
  await page.screenshot({ path: path.join(__dirname, 'login-profile-first-connection.png') })

  expect(errors).toEqual([])
})

test('layout remains usable at a compact desktop viewport', async ({ page }) => {
  const errors = []
  page.on('pageerror', (error) => errors.push(error.message))

  await page.setViewportSize({ width: 1024, height: 768 })
  await page.goto(`${prototypeURL}?view=first`)
  await expect(page.getByLabel('Profile 名称')).toBeVisible()
  await expect(page.getByRole('button', { name: /提交申请/ })).toBeVisible()
  expect(errors).toEqual([])
})
