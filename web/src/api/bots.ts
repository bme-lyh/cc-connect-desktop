import api from './client';

export interface AgentHealth {
  key: string;
  label: string;
  installed: boolean;
  logged_in: boolean;
  version?: string;
  path?: string;
  problem?: string;
  guide?: string;
}

export interface SetupStatus {
  first_run: boolean;
  bot_count: number;
  agents: AgentHealth[];
}

export interface SetupField {
  key: string;
  label: string;
  required?: boolean;
  type?: 'text' | 'password' | 'number' | 'boolean' | 'select';
  secret?: boolean;
  placeholder?: string;
  hint?: string;
  group?: 'basic' | 'advanced';
  options?: string[];
  show_when?: Record<string, string[]>;
}

export interface SetupPlatform { key: string; label: string; qr?: boolean; fields: SetupField[]; }
export interface SetupAgent { key: string; label: string; recommended?: boolean; modes?: string[]; }
export interface SetupCatalog { agents: SetupAgent[]; platforms: SetupPlatform[]; }
export interface SetupModel { name: string; description?: string; alias?: string; }
export interface SetupModelCatalog { models: SetupModel[]; current?: string; }
export interface DirectorySelection { path: string; cancelled: boolean; }

export interface BotSummary {
  id: string;
  name: string;
  display_name: string;
  enabled: boolean;
  agent_type: string;
  work_dir: string;
  permission_mode?: string;
  model?: string;
  reasoning_effort?: string;
  reply_footer: boolean;
  platform_type: string;
  configured: boolean;
  runtime_state: 'running' | 'stopped' | 'error' | 'applying';
  runtime_error?: string;
}

export interface BotRequest {
  id?: string;
  display_name: string;
  name?: string;
  enabled?: boolean;
  agent_type: string;
  work_dir: string;
  permission_mode?: string;
  model?: string;
  reasoning_effort?: string;
  reply_footer?: boolean;
  platform_type: string;
  options: Record<string, unknown>;
}

export const getSetupStatus = () => api.get<SetupStatus>('/setup/status');
export const getSetupCatalog = () => api.get<SetupCatalog>('/setup/catalog');
export const getSetupModels = (agent: string) => api.get<SetupModelCatalog>('/setup/models', { agent });
export const selectSetupDirectory = () => api.post<DirectorySelection>('/setup/select-directory');
export const listBots = () => api.get<{ bots: BotSummary[] }>('/bots');
export const createBot = (body: BotRequest) => api.post<{ bot: BotSummary; applying: boolean }>('/bots', body);
export const updateBot = (id: string, body: BotRequest) => api.put<{ bot: BotSummary; applying: boolean }>(`/bots/${id}`, body);
export const updateBotState = (id: string, enabled: boolean) => api.patch(`/bots/${id}/state`, { enabled });
export const removeBot = (id: string) => api.delete(`/bots/${id}`);
export const ready = () => api.get<{ ready: boolean }>('/ready');

export async function waitUntilReady(timeoutMs = 20_000): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  await new Promise(resolve => setTimeout(resolve, 700));
  while (Date.now() < deadline) {
    try {
      const status = await ready();
      if (status.ready) return;
    } catch {
      // A short connection failure is expected while the process restarts.
    }
    await new Promise(resolve => setTimeout(resolve, 300));
  }
  throw new Error('cc-connect did not become ready after applying the configuration');
}
