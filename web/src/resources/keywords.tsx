import { useState, useEffect, useCallback } from 'react';
import { useDataProvider, useNotify, Title } from 'react-admin';
import Box from '@mui/material/Box';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import Typography from '@mui/material/Typography';
import TextField from '@mui/material/TextField';
import Select from '@mui/material/Select';
import MenuItem from '@mui/material/MenuItem';
import IconButton from '@mui/material/IconButton';
import Button from '@mui/material/Button';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/DeleteOutline';
import SaveIcon from '@mui/icons-material/Save';
import Chip from '@mui/material/Chip';
import { saveKeywords } from '../dataProvider';

interface Keyword {
  keyword: string;
  mode: string;
}

const modeColors: Record<string, 'error' | 'warning' | 'info'> = {
  do: 'error',
  plan: 'warning',
  ask: 'info',
};

export const KeywordEditor = ({ projectId }: { projectId: string }) => {
  const dataProvider = useDataProvider();
  const notify = useNotify();
  const [keywords, setKeywords] = useState<Keyword[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const loadKeywords = useCallback(async () => {
    try {
      const { data } = await dataProvider.getList('keywords', {
        pagination: { page: 1, perPage: 100 },
        sort: { field: 'keyword', order: 'ASC' },
        filter: { projectId },
      });
      setKeywords(data.map((d: Record<string, unknown>) => ({
        keyword: String(d.keyword || ''),
        mode: String(d.mode || 'ask'),
      })));
    } catch {
      notify('Failed to load keywords', { type: 'error' });
    } finally {
      setLoading(false);
    }
  }, [dataProvider, projectId, notify]);

  useEffect(() => { loadKeywords(); }, [loadKeywords]);

  const handleSave = async () => {
    setSaving(true);
    try {
      await saveKeywords(projectId, keywords.filter(k => k.keyword.trim()));
      notify('Keywords saved', { type: 'success' });
    } catch {
      notify('Failed to save keywords', { type: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const addRow = () => setKeywords([...keywords, { keyword: '', mode: 'ask' }]);

  const removeRow = (index: number) => {
    const next = [...keywords];
    next.splice(index, 1);
    setKeywords(next);
  };

  const updateRow = (index: number, field: keyof Keyword, value: string) => {
    const next = [...keywords];
    next[index] = { ...next[index], [field]: value };
    setKeywords(next);
  };

  if (loading) return <Typography>Loading…</Typography>;

  return (
    <Box>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Keyword</TableCell>
            <TableCell sx={{ width: 140 }}>Mode</TableCell>
            <TableCell sx={{ width: 60 }} />
          </TableRow>
        </TableHead>
        <TableBody>
          {keywords.map((kw, i) => (
            <TableRow key={i}>
              <TableCell>
                <TextField
                  value={kw.keyword}
                  onChange={(e) => updateRow(i, 'keyword', e.target.value)}
                  size="small"
                  fullWidth
                  placeholder="e.g. @opencode"
                  sx={{ '& input': { fontFamily: '"JetBrains Mono", monospace' } }}
                />
              </TableCell>
              <TableCell>
                <Select
                  value={kw.mode}
                  onChange={(e) => updateRow(i, 'mode', e.target.value)}
                  size="small"
                  fullWidth
                >
                  <MenuItem value="ask"><Chip label="ask" size="small" color="info" variant="outlined" /></MenuItem>
                  <MenuItem value="plan"><Chip label="plan" size="small" color="warning" variant="outlined" /></MenuItem>
                  <MenuItem value="do"><Chip label="do" size="small" color="error" variant="outlined" /></MenuItem>
                </Select>
              </TableCell>
              <TableCell>
                <IconButton size="small" onClick={() => removeRow(i)} color="error">
                  <DeleteIcon fontSize="small" />
                </IconButton>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      <Box sx={{ display: 'flex', gap: 2, mt: 2 }}>
        <Button startIcon={<AddIcon />} onClick={addRow} variant="outlined" size="small">
          Add Keyword
        </Button>
        <Button startIcon={<SaveIcon />} onClick={handleSave} variant="contained" size="small" disabled={saving}>
          {saving ? 'Saving…' : 'Save All'}
        </Button>
      </Box>
    </Box>
  );
};

const KeywordsPage = () => {
  const [projectId, setProjectId] = useState('');
  const [activeProject, setActiveProject] = useState('');

  return (
    <Box sx={{ p: 3 }}>
      <Title title="Trigger Keywords" />
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" sx={{ mb: 2 }}>Select Project</Typography>
          <Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
            <TextField
              label="Project ID"
              value={projectId}
              onChange={(e) => setProjectId(e.target.value)}
              size="small"
            />
            <Button variant="contained" onClick={() => setActiveProject(projectId)} disabled={!projectId}>
              Load Keywords
            </Button>
          </Box>
        </CardContent>
      </Card>
      {activeProject && (
        <Card>
          <CardContent>
            <Typography variant="h6" sx={{ mb: 2 }}>
              Keywords for project: <Chip label={activeProject} size="small" />
            </Typography>
            <KeywordEditor projectId={activeProject} />
          </CardContent>
        </Card>
      )}
    </Box>
  );
};

export default KeywordsPage;
