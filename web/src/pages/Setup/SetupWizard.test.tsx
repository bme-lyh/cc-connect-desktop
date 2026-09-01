import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { listBots } from '@/api/bots';
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
  getSetupModels: vi.fn().mockResolvedValue({
    models: [
      { name: 'gpt-5.6-sol', description: 'Frontier coding model' },
      { name: 'gpt-5.6-terra' },
    ],
    current: 'gpt-5.6-sol',
  }),
  selectSetupDirectory: vi.fn().mockResolvedValue({ path: 'D:\\projects\\selected', cancelled: false }),
  listBots: vi.fn().mockResolvedValue({ bots: [] }),
  createBot: vi.fn(),
  updateBot: vi.fn(),
  waitUntilReady: vi.fn(),
}));

vi.mock('@/api/setup', () => ({
  setupFeishuBegin: vi.fn(), setupFeishuPoll: vi.fn(), setupWeixinBegin: vi.fn(), setupWeixinPoll: vi.fn(),
}));

afterEach(cleanup);

describe('SetupWizard', () => {
  it('starts without a project and surfaces local Codex and Pi diagnostics', async () => {
    render(<MemoryRouter><SetupWizard /></MemoryRouter>);
    expect(await screen.findByText('Codex')).toBeInTheDocument();
    expect(screen.getByText('Pi')).toBeInTheDocument();
    expect(screen.getByText('codex-cli 0.152.0')).toBeInTheDocument();
    expect(screen.getByText('0.84.4')).toBeInTheDocument();
    expect(screen.queryByText('Gemini CLI')).not.toBeInTheDocument();
  });

  it('uses the native directory picker and an agent-provided model list', async () => {
    render(<MemoryRouter><SetupWizard /></MemoryRouter>);
    await screen.findByText('codex-cli 0.152.0');

    fireEvent.click(screen.getByRole('button', { name: /next/i }));
    fireEvent.change(screen.getByPlaceholderText('Development assistant'), { target: { value: 'Development bot' } });
    fireEvent.click(screen.getByRole('button', { name: /next/i }));

    expect(await screen.findByRole('option', { name: /gpt-5.6-sol/ })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'gpt-5.6-terra' })).toBeInTheDocument();
    await waitFor(() => expect(screen.getByLabelText('Model')).toHaveValue('gpt-5.6-sol'));
    expect(screen.getByLabelText('Show answer details')).not.toBeChecked();

    fireEvent.click(screen.getByRole('button', { name: 'Choose folder' }));
    await waitFor(() => expect(screen.getByRole('textbox')).toHaveValue('D:\\projects\\selected'));
  });

  it('preserves an existing model that is absent from local discovery', async () => {
    vi.mocked(listBots).mockResolvedValueOnce({
      bots: [{
        id: 'bot-1', name: 'bot-1', display_name: 'Legacy bot', enabled: true,
        agent_type: 'codex', work_dir: 'D:\\projects\\legacy', permission_mode: 'suggest',
        model: 'legacy-model', reasoning_effort: 'medium', reply_footer: true,
        platform_type: 'telegram', configured: true, runtime_state: 'stopped',
      }],
    });

    render(<MemoryRouter initialEntries={['/?edit=bot-1']}><SetupWizard /></MemoryRouter>);
    await screen.findByText('codex-cli 0.152.0');
    fireEvent.click(screen.getByRole('button', { name: /next/i }));
    fireEvent.click(screen.getByRole('button', { name: /next/i }));

    expect(await screen.findByRole('option', { name: 'legacy-model' })).toBeInTheDocument();
    await waitFor(() => expect(screen.getByLabelText('Model')).toHaveValue('legacy-model'));
  });
});
