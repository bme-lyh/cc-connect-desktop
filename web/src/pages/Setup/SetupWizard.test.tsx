import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import SetupWizard from './SetupWizard';

vi.mock('@/api/bots', () => ({
  getSetupCatalog: vi.fn().mockResolvedValue({
    agents: [
      { key: 'codex', label: 'Codex', recommended: true },
      { key: 'pi', label: 'Pi', recommended: true },
      { key: 'gemini', label: 'Gemini CLI' },
    ],
    platforms: [
      { key: 'feishu', label: 'Feishu', qr: true, fields: [] },
      { key: 'telegram', label: 'Telegram', fields: [{ key: 'token', label: 'fields.botToken', required: true, secret: true }] },
    ],
  }),
  getSetupStatus: vi.fn().mockResolvedValue({
    first_run: true,
    bot_count: 0,
    agents: [
      { key: 'codex', label: 'Codex', installed: true, logged_in: true, version: 'codex-cli 0.152.0' },
      { key: 'pi', label: 'Pi', installed: true, logged_in: true, version: '0.84.4' },
    ],
  }),
  createBot: vi.fn(),
  waitUntilReady: vi.fn(),
}));

vi.mock('@/api/setup', () => ({
  setupFeishuBegin: vi.fn(), setupFeishuPoll: vi.fn(), setupWeixinBegin: vi.fn(), setupWeixinPoll: vi.fn(),
}));

describe('SetupWizard', () => {
  it('starts without a project and surfaces local Codex and Pi diagnostics', async () => {
    render(<MemoryRouter><SetupWizard /></MemoryRouter>);
    expect(await screen.findByText('Codex')).toBeInTheDocument();
    expect(screen.getByText('Pi')).toBeInTheDocument();
    expect(screen.getByText('codex-cli 0.152.0')).toBeInTheDocument();
    expect(screen.getByText('0.84.4')).toBeInTheDocument();
    expect(screen.queryByText('Gemini CLI')).not.toBeInTheDocument();
  });
});
