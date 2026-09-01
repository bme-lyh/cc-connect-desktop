import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import BotList from './BotList';

const listBots = vi.fn();
vi.mock('@/api/bots', () => ({
  listBots: (...args: unknown[]) => listBots(...args),
  updateBotState: vi.fn(),
  removeBot: vi.fn(),
  waitUntilReady: vi.fn(),
}));

describe('BotList', () => {
  beforeEach(() => {
    listBots.mockResolvedValue({
      bots: [{
        id: 'bot-id', name: 'bot-project', display_name: 'Development assistant', enabled: true,
        agent_type: 'codex', work_dir: 'D:\\work', permission_mode: 'default', model: 'gpt-test',
        platform_type: 'telegram', configured: true, runtime_state: 'running',
      }],
    });
  });

  it('renders a concise card with runtime and binding details', async () => {
    render(<MemoryRouter><BotList /></MemoryRouter>);
    expect(await screen.findByText('Development assistant')).toBeInTheDocument();
    expect(screen.getByText('telegram · codex')).toBeInTheDocument();
    expect(screen.getByText('gpt-test')).toBeInTheDocument();
    expect(screen.getByText('Running')).toBeInTheDocument();
  });
});
