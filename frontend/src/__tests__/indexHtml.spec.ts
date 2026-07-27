import { resolve } from 'node:path';
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const INDEX_HTML_PATH = resolve(__dirname, '../../index.html');

describe('frontend/index.html', () => {
  it('contains <html lang="en">', () => {
    const html = readIndexHtml();
    expect(html).toContain('<html lang="en">');
  });

  it('meta description does NOT contain third-party AI brand names', () => {
    const html = readIndexHtml();
    const metaDescription = extractMetaDescription(html);
    expect(metaDescription).not.toMatch(/Claude|OpenAI|Gemini/i);
  });

  it('meta description contains "AI API gateway"', () => {
    const html = readIndexHtml();
    const metaDescription = extractMetaDescription(html);
    expect(metaDescription).toContain('AI API gateway');
  });
});

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function readIndexHtml(): string {
  return readFileSync(INDEX_HTML_PATH, 'utf-8');
}

function extractMetaDescription(html: string): string {
  const match = html.match(
    /<meta\s+name=["']description["']\s+content=["']([^"']+)["']/i,
  );
  if (!match) {
    throw new Error('meta description tag not found in index.html');
  }
  return match[1];
}
