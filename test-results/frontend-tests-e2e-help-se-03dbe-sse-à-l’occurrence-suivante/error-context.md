# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: frontend/tests/e2e/help-search.spec.js >> « / » cherche dans l’aide, Entrée passe à l’occurrence suivante
- Location: frontend/tests/e2e/help-search.spec.js:12:1

# Error details

```
Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
Call log:
  - navigating to "/", waiting until "load"

```

# Test source

```ts
  1  | /**
  2  |  * help-search.spec.js — la recherche dans l'aide
  3  |  *
  4  |  * « ? » ouvre l'aide, « / » met le focus dans le champ de recherche sans la fermer, la saisie
  5  |  * compte les occurrences et Entrée passe à la suivante.
  6  |  */
  7  | 
  8  | import { test, expect } from '@playwright/test';
  9  | import { installWailsMock } from './helpers/wailsMock.js';
  10 | import { openLibraryMock } from './helpers/fixtures.js';
  11 | 
  12 | test('« / » cherche dans l’aide, Entrée passe à l’occurrence suivante', async ({ page }) => {
  13 |     await installWailsMock(page, openLibraryMock());
> 14 |     await page.goto('/');
     |                ^ Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
  15 |     await expect(page.getByTestId('status-bar')).toContainText('3 / 3');
  16 | 
  17 |     await page.keyboard.press('?');
  18 |     const search = page.getByTestId('help-search');
  19 |     await expect(search).toBeVisible();
  20 |     await expect(page.getByTestId('help-loading')).toHaveCount(0);
  21 | 
  22 |     await page.keyboard.press('/');
  23 |     await expect(search, `focus is on ${await page.evaluate(() => document.activeElement?.outerHTML.slice(0, 160))}`).toBeFocused();
  24 |     await search.pressSequentially('position');
  25 |     await expect(search).toHaveValue('position');
  26 |     await expect(page.getByTestId('help-search-count')).toHaveText(/^0 \/ \d+$/);
  27 |     await search.press('Enter');
  28 |     await expect(page.getByTestId('help-search-count')).toHaveText(/^1 \/ \d+$/);
  29 |     await search.press('Enter');
  30 |     await expect(page.getByTestId('help-search-count')).toHaveText(/^2 \/ \d+$/);
  31 |     await expect(search).toBeVisible();
  32 | });
  33 | 
  34 | test('un clic dans le champ de l’aide y met le focus et la frappe le garde', async ({ page }) => {
  35 |     await installWailsMock(page, openLibraryMock());
  36 |     await page.goto('/');
  37 |     await expect(page.getByTestId('status-bar')).toContainText('3 / 3');
  38 | 
  39 |     await page.keyboard.press('?');
  40 |     const search = page.getByTestId('help-search');
  41 |     await expect(search).toBeVisible();
  42 |     await expect(page.getByTestId('help-loading')).toHaveCount(0);
  43 | 
  44 |     await search.click();
  45 |     await expect(search).toBeFocused();
  46 |     await page.keyboard.type('position');
  47 |     await expect(search).toHaveValue('position');
  48 |     await expect(search).toBeFocused();
  49 |     await page.waitForTimeout(400);
  50 |     await expect(search).toBeFocused();
  51 | });
  52 | 
```