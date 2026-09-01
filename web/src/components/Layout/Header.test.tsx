import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import i18n from '@/i18n';
import { shutdownSystem } from '@/api/status';
import Header from './Header';

vi.mock('@/api/status', () => ({
  shutdownSystem: vi.fn().mockResolvedValue({}),
}));

describe('Header shutdown', () => {
  beforeEach(async () => {
    await i18n.changeLanguage('en');
    vi.mocked(shutdownSystem).mockClear();
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it('confirms and gracefully stops the local application', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    render(<Header />);

    fireEvent.click(screen.getByRole('button', { name: 'Quit cc-connect' }));

    await waitFor(() => expect(shutdownSystem).toHaveBeenCalledTimes(1));
    expect(await screen.findByText('cc-connect has stopped')).toBeInTheDocument();
    expect(screen.getByText(/close this browser tab/i)).toBeInTheDocument();
  });

  it('does not stop the application when confirmation is cancelled', () => {
    vi.spyOn(window, 'confirm').mockReturnValue(false);
    render(<Header />);

    fireEvent.click(screen.getByRole('button', { name: 'Quit cc-connect' }));

    expect(shutdownSystem).not.toHaveBeenCalled();
  });
});
