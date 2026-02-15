import { useEffect, useState } from 'react';
import { useDataProvider, useRedirect, Title } from 'react-admin';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Button from '@mui/material/Button';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Chip from '@mui/material/Chip';
import FolderIcon from '@mui/icons-material/FolderOpen';
import TaskIcon from '@mui/icons-material/Assignment';
import AddIcon from '@mui/icons-material/Add';
import SettingsIcon from '@mui/icons-material/Tune';

interface Task {
  id: string | number;
  status: string;
  title: string;
  provider_type: string;
  trigger_mode: string;
  created_at: string;
}

const statusColors: Record<string, 'success' | 'warning' | 'info' | 'error' | 'default'> = {
  completed: 'success',
  processing: 'warning',
  pending: 'info',
  failed: 'error',
};

const StatCard = ({ icon, label, value, gradient }: { icon: React.ReactNode; label: string; value: number | string; gradient: string }) => (
  <Card sx={{
    flex: 1,
    minWidth: 200,
    background: gradient,
    border: 'none',
    position: 'relative',
    overflow: 'hidden',
    '&::after': {
      content: '""',
      position: 'absolute',
      top: 0,
      right: 0,
      width: '50%',
      height: '100%',
      background: 'radial-gradient(circle at 80% 50%, rgba(255,255,255,0.06) 0%, transparent 70%)',
    },
  }}>
    <CardContent sx={{ p: 3, '&:last-child': { pb: 3 } }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 1.5 }}>
        <Box sx={{ opacity: 0.8 }}>{icon}</Box>
        <Typography variant="overline" sx={{ color: 'rgba(255,255,255,0.7)', lineHeight: 1 }}>
          {label}
        </Typography>
      </Box>
      <Typography variant="h3" sx={{ color: '#fff', fontWeight: 700, lineHeight: 1 }}>
        {value}
      </Typography>
    </CardContent>
  </Card>
);

const Dashboard = () => {
  const dataProvider = useDataProvider();
  const redirect = useRedirect();
  const [projectCount, setProjectCount] = useState<number>(0);
  const [taskCount, setTaskCount] = useState<number>(0);
  const [recentTasks, setRecentTasks] = useState<Task[]>([]);

  useEffect(() => {
    dataProvider.getList('projects', { pagination: { page: 1, perPage: 1 }, sort: { field: 'id', order: 'ASC' }, filter: {} })
      .then(({ total }) => setProjectCount(total ?? 0))
      .catch(() => {});

    dataProvider.getList('tasks', { pagination: { page: 1, perPage: 10 }, sort: { field: 'created_at', order: 'DESC' }, filter: {} })
      .then(({ data, total }) => {
        setTaskCount(total ?? 0);
        setRecentTasks(data as Task[]);
      })
      .catch(() => {});
  }, [dataProvider]);

  return (
    <Box sx={{ p: { xs: 2, md: 3 } }}>
      <Title title="Dashboard" />

      <Box sx={{ mb: 4 }}>
        <Typography variant="h4" sx={{ mb: 0.5 }}>OpenCode Bot</Typography>
        <Typography variant="body2" color="text.secondary">
          System overview and quick actions
        </Typography>
      </Box>

      <Box sx={{ display: 'flex', gap: 2, mb: 4, flexWrap: 'wrap' }}>
        <StatCard
          icon={<FolderIcon sx={{ color: '#fff', fontSize: 28 }} />}
          label="Projects"
          value={projectCount}
          gradient="linear-gradient(135deg, #059669 0%, #34d399 100%)"
        />
        <StatCard
          icon={<TaskIcon sx={{ color: '#fff', fontSize: 28 }} />}
          label="Total Tasks"
          value={taskCount}
          gradient="linear-gradient(135deg, #4f46e5 0%, #818cf8 100%)"
        />
      </Box>

      <Box sx={{ display: 'flex', gap: 2, mb: 4, flexWrap: 'wrap' }}>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => redirect('/projects/create')}
          size="large"
        >
          Add Project
        </Button>
        <Button
          variant="outlined"
          startIcon={<SettingsIcon />}
          onClick={() => redirect('/settings')}
          size="large"
        >
          Settings
        </Button>
      </Box>

      <Card>
        <CardContent sx={{ p: 0, '&:last-child': { pb: 0 } }}>
          <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
            <Typography variant="h6">Recent Tasks</Typography>
          </Box>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Status</TableCell>
                <TableCell>Provider</TableCell>
                <TableCell>Title</TableCell>
                <TableCell>Mode</TableCell>
                <TableCell>Created</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {recentTasks.length === 0 && (
                <TableRow>
                  <TableCell colSpan={5} align="center" sx={{ py: 4, color: 'text.secondary' }}>
                    No tasks yet
                  </TableCell>
                </TableRow>
              )}
              {recentTasks.map((task) => (
                <TableRow
                  key={task.id}
                  hover
                  sx={{ cursor: 'pointer' }}
                  onClick={() => redirect(`/tasks/${task.id}/show`)}
                >
                  <TableCell>
                    <Chip
                      label={task.status}
                      color={statusColors[task.status] || 'default'}
                      size="small"
                      variant="outlined"
                    />
                  </TableCell>
                  <TableCell>
                    <Chip label={task.provider_type} size="small" variant="filled" />
                  </TableCell>
                  <TableCell>{task.title || '—'}</TableCell>
                  <TableCell>
                    <Typography variant="caption">{task.trigger_mode || '—'}</Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="caption">
                      {task.created_at ? new Date(task.created_at).toLocaleString() : '—'}
                    </Typography>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </Box>
  );
};

export default Dashboard;
