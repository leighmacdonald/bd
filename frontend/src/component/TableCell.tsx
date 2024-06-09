import { PropsWithChildren } from 'react';
import Typography from '@mui/material/Typography';

export const TableHeading = ({ children }: PropsWithChildren) => {
    return (
        <Typography align={'left'} padding={0} fontWeight={500}>
            {children}
        </Typography>
    );
};
