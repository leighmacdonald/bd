import { inputValidator } from '../util.ts';
import { ChangeEvent, useCallback, useState } from 'react';
import Tooltip from '@mui/material/Tooltip';
import { FormControl, FormHelperText, TextField } from '@mui/material';

interface SettingsTextBoxProps {
    label: string;
    value: string;
    setValue: (value: string) => void;
    tooltip: string;
    secrets?: boolean;
    validator?: inputValidator;
    disabled?: boolean;
}

const SettingsTextBox = ({
    label,
    value,
    setValue,
    tooltip,
    secrets,
    validator,
    disabled
}: SettingsTextBoxProps) => {
    const [error, setError] = useState<string | null>(null);
    const handleChange = useCallback(
        (event: ChangeEvent<HTMLTextAreaElement>) => {
            setValue(event.target.value);
            if (validator) {
                setError(validator(event.target.value));
            }
        },
        [setValue, validator]
    );

    return (
        <Tooltip title={tooltip} placement={'top'}>
            <FormControl fullWidth>
                <TextField
                    disabled={disabled}
                    hiddenLabel
                    type={secrets ? 'password' : 'text'}
                    error={Boolean(error)}
                    id={`settings-textfield-${label}`}
                    label={label}
                    value={value}
                    onChange={handleChange}
                />
                <FormHelperText>{error}</FormHelperText>
            </FormControl>
        </Tooltip>
    );
};

export default SettingsTextBox;
