'use client';

import { Button, ButtonGroup, Flex, HStack, Input, Text } from '@chakra-ui/react';
import { useEffect, useState } from 'react';
import { DateRangeFilter } from '@/lib/types';

interface DateRangeFilterProps {
  value?: DateRangeFilter | null;
  onChange?: (value: DateRangeFilter | null) => void;
}

const QUICK_RANGES = [
  { label: '1h', minutes: 60 },
  { label: '6h', minutes: 360 },
  { label: '24h', minutes: 1440 },
  { label: '7d', minutes: 7 * 24 * 60 },
  { label: '30d', minutes: 30 * 24 * 60 }
];

export function DateRangeFilterControl({ value, onChange }: DateRangeFilterProps) {
  const [from, setFrom] = useState(value?.from ? value.from.slice(0, 16) : '');
  const [to, setTo] = useState(value?.to ? value.to.slice(0, 16) : '');

  useEffect(() => {
    setFrom(value?.from ? value.from.slice(0, 16) : '');
    setTo(value?.to ? value.to.slice(0, 16) : '');
  }, [value?.from, value?.to]);

  const apply = () => {
    if (onChange) {
      if (!from && !to) {
        onChange(null);
      } else {
        onChange({
          from: from ? new Date(from).toISOString() : null,
          to: to ? new Date(to).toISOString() : null
        });
      }
    }
  };

  const clear = () => {
    setFrom('');
    setTo('');
    onChange?.(null);
  };

  const applyQuick = (minutes: number) => {
    const now = new Date();
    const past = new Date(now.getTime() - minutes * 60 * 1000);
    const fromISO = past.toISOString();
    const toISO = now.toISOString();
    setFrom(fromISO.slice(0, 16));
    setTo(toISO.slice(0, 16));
    onChange?.({ from: fromISO, to: toISO });
  };

  return (
    <Flex direction={{ base: 'column', xl: 'row' }} gap={4} align={{ base: 'flex-start', xl: 'center' }}>
      <HStack spacing={3} w={{ base: '100%', xl: 'auto' }}>
        <Input
          type="datetime-local"
          value={from}
          onChange={event => setFrom(event.target.value)}
          bg="white"
          borderColor="gray.200"
          _hover={{ borderColor: 'gray.300' }}
        />
        <Text color="gray.500">to</Text>
        <Input
          type="datetime-local"
          value={to}
          onChange={event => setTo(event.target.value)}
          bg="white"
          borderColor="gray.200"
          _hover={{ borderColor: 'gray.300' }}
        />
      </HStack>
      <ButtonGroup size="sm" variant="outline" colorScheme="gray">
        {QUICK_RANGES.map(range => (
          <Button key={range.label} onClick={() => applyQuick(range.minutes)}>
            {range.label}
          </Button>
        ))}
      </ButtonGroup>
      <HStack spacing={3}>
        <Button size="sm" colorScheme="blue" onClick={apply}>
          Apply
        </Button>
        <Button size="sm" variant="outline" onClick={clear}>
          Clear
        </Button>
      </HStack>
    </Flex>
  );
}
