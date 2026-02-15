import {
  List, Datagrid, TextField, DateField, BooleanField,
  DeleteButton, EditButton,
  Create, Edit, SimpleForm, TextInput, SelectInput, BooleanInput, PasswordInput,
  Show, SimpleShowLayout,
  usePermissions, FunctionField,
} from 'react-admin';
import Chip from '@mui/material/Chip';

const roleChoices = [
  { id: 'admin', name: 'Admin' },
  { id: 'editor', name: 'Editor' },
  { id: 'viewer', name: 'Viewer' },
];

const roleColors: Record<string, 'error' | 'warning' | 'info'> = {
  admin: 'error',
  editor: 'warning',
  viewer: 'info',
};

export const UserList = () => {
  const { permissions } = usePermissions();
  return (
    <List sort={{ field: 'username', order: 'ASC' }}>
      <Datagrid bulkActionButtons={false}>
        <TextField source="username" />
        <TextField source="display_name" label="Display Name" />
        <FunctionField
          label="Role"
          render={(record: Record<string, unknown>) => (
            <Chip
              label={String(record.role || 'viewer').toUpperCase()}
              size="small"
              color={roleColors[String(record.role)] || 'default'}
              variant="filled"
            />
          )}
        />
        <BooleanField source="enabled" />
        <DateField source="created_at" label="Created" showTime />
        {permissions === 'admin' && <EditButton />}
        {permissions === 'admin' && <DeleteButton />}
      </Datagrid>
    </List>
  );
};

export const UserCreate = () => (
  <Create redirect="list">
    <SimpleForm>
      <TextInput source="username" fullWidth isRequired />
      <PasswordInput source="password" fullWidth isRequired />
      <TextInput source="display_name" label="Display Name" fullWidth />
      <SelectInput source="role" choices={roleChoices} isRequired fullWidth defaultValue="viewer" />
    </SimpleForm>
  </Create>
);

export const UserEdit = () => (
  <Edit>
    <SimpleForm>
      <TextInput source="username" fullWidth disabled />
      <TextInput source="display_name" label="Display Name" fullWidth />
      <SelectInput source="role" choices={roleChoices} isRequired fullWidth />
      <BooleanInput source="enabled" />
    </SimpleForm>
  </Edit>
);

export const UserShow = () => (
  <Show>
    <SimpleShowLayout>
      <TextField source="id" />
      <TextField source="username" />
      <TextField source="display_name" label="Display Name" />
      <FunctionField
        label="Role"
        render={(record: Record<string, unknown>) => (
          <Chip
            label={String(record.role || 'viewer').toUpperCase()}
            color={roleColors[String(record.role)] || 'default'}
            variant="filled"
          />
        )}
      />
      <BooleanField source="enabled" />
      <DateField source="created_at" label="Created" showTime />
    </SimpleShowLayout>
  </Show>
);
