// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import { Button, Checkbox, MantineProvider, Radio, Textarea, TextInput } from "@mantine/core";
import { certifyRoots, LIBRARY_PRIMITIVES } from "./roots";

afterEach(cleanup);
it.each(LIBRARY_PRIMITIVES)("certifies $name only through its declared module import", p => {
  const [name, ...members] = p.name.split('.');
  const target = ['Control', ...members].join('.');
  const prop = Object.keys(p.callbacks)[0];
  const source = (module: string) => `import {${name} as Control} from '${module}'; function App(){return <${target} ${prop}={()=>{}} />}`;
  const good = certifyRoots(new Map([['fixture.tsx',source(p.module)]]));
  expect(good.roots).toHaveLength(1);
  expect(good.roots[0].event).toBe(Object.values(p.callbacks)[0]);
  const unknown = certifyRoots(new Map([['fixture.tsx',source('unverified-library')]]));
  expect(unknown.roots).toHaveLength(0);
  expect(unknown.uncertified).toHaveLength(1);
});
it.each(LIBRARY_PRIMITIVES)("backs $name callbacks with the installed styled component DOM", p => {
  const callback = vi.fn();
  const control = p.name === 'Button' ? <Button onClick={callback}>Fixture</Button>
    : p.name === 'Checkbox' ? <Checkbox label="Fixture" onChange={callback} />
    : p.name === 'TextInput' ? <TextInput aria-label="Fixture" onChange={callback} />
    : p.name === 'Textarea' ? <Textarea aria-label="Fixture" onChange={callback} />
    : p.name === 'Radio.Card' ? <Radio.Card onClick={callback}>Fixture</Radio.Card>
    : <Checkbox.Card onClick={callback}>Fixture</Checkbox.Card>;
  render(<MantineProvider forceColorScheme="light">{control}</MantineProvider>);
  const role = p.name === 'Button' ? 'button' : p.name === 'Radio.Card' ? 'radio' : p.name.startsWith('Checkbox') ? 'checkbox' : 'textbox';
  const input = screen.getByRole(role,{name:'Fixture'});
  if(role === 'textbox') fireEvent.change(input,{target:{value:'value'}});else fireEvent.click(input);
  expect(callback).toHaveBeenCalledTimes(1);
});
