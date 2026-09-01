import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import { ArrowRight, FolderKanban, Server } from 'lucide-react';
import { Badge, Card, EmptyState } from '@/components/ui';
import { listProjects, type ProjectSummary } from '@/api/projects';

// Creation lives in the catalog-driven setup wizard. This page remains the
// advanced runtime view and intentionally carries no agent/platform metadata.
export default function ProjectList() {
  const { t } = useTranslation();
  const [projects, setProjects] = useState<ProjectSummary[]>([]);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    try {
      const result = await listProjects();
      setProjects(result.projects || []);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { refresh(); }, [refresh]);

  if (loading) return <div className="h-64 flex items-center justify-center text-gray-400">Loading...</div>;

  return (
    <div className="space-y-4 animate-fade-in">
      <div>
        <h2 className="text-lg font-bold text-gray-900 dark:text-white">{t('projects.title')}</h2>
        <p className="text-xs text-gray-500 mt-1">{t('simple.advancedProjectsHint')}</p>
      </div>
      {projects.length === 0 ? <EmptyState message={t('projects.noProjects')} icon={FolderKanban} /> : (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          {projects.map(project => (
            <Link key={project.name} to={`/projects/${project.name}`}>
              <Card hover className="h-full">
                <div className="flex items-start justify-between mb-3">
                  <div className="flex items-center gap-2"><Server size={18} className="text-gray-400" /><h3 className="font-semibold text-gray-900 dark:text-white">{project.name}</h3></div>
                  <ArrowRight size={16} className="text-gray-300" />
                </div>
                <div className="flex flex-wrap gap-1.5"><Badge variant="info">{project.agent_type}</Badge>{project.platforms?.map(platform => <Badge key={platform}>{platform}</Badge>)}</div>
              </Card>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
