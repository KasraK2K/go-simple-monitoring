'use client';

import { Badge, Button, HStack, Icon, Menu, MenuButton, MenuItem, MenuList, Text } from '@chakra-ui/react';
import { FiChevronDown, FiCloud, FiServer } from 'react-icons/fi';
import { MonitorServerOption } from '@/lib/types';

interface ServerSwitcherProps {
  servers: MonitorServerOption[];
  activeServer?: MonitorServerOption | null;
  onSelect: (server: MonitorServerOption | null) => void;
}

const LOCAL_OPTION: MonitorServerOption = { name: 'Local Instance', address: '' };

export function ServerSwitcher({ servers, activeServer, onSelect }: ServerSwitcherProps) {
  const items = [LOCAL_OPTION, ...servers];
  const current = activeServer?.address ? activeServer : LOCAL_OPTION;

  return (
    <Menu>
      <MenuButton
        as={Button}
        rightIcon={<FiChevronDown />}
        variant="outline"
        colorScheme="blue"
        borderColor="gray.200"
        bg="white"
        _hover={{ bg: 'gray.50' }}
      >
        <HStack spacing={2}>
          <Icon as={current.address ? FiCloud : FiServer} />
          <Text>{current.name}</Text>
          {current.address ? (
            <Badge colorScheme="cyan" fontSize="0.65rem">
              Remote
            </Badge>
          ) : null}
        </HStack>
      </MenuButton>
      <MenuList minW="56">
        {items.map(server => (
          <MenuItem
            key={server.address || 'local'}
            onClick={() => onSelect(server.address ? server : null)}
            icon={<Icon as={server.address ? FiCloud : FiServer} />}
            _hover={{ bg: 'gray.50' }}
          >
            <HStack justify="space-between" w="full">
              <Text>{server.name}</Text>
              {current.address === server.address && current.name === server.name ? (
                <Badge colorScheme="green">Active</Badge>
              ) : null}
            </HStack>
          </MenuItem>
        ))}
      </MenuList>
    </Menu>
  );
}
