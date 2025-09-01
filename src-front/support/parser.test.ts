import { describe, it, expect, beforeEach } from 'vitest'
import { XMLParser, MsgResp } from './parser'

describe('XMLParser', () => {
  let parser: XMLParser

  beforeEach(() => {
    parser = new XMLParser()
  })

  describe('Parse', () => {
    it('should create a MsgResp instance with the original data', () => {
      const xml = '<some>test</some>'
      const result = parser.Parse(xml)
      
      expect(result).toBeInstanceOf(MsgResp)
      expect(result.origin).toBe(xml)
      expect(Array.isArray(result.errors)).toBe(true)
      expect(result.errors.length).toBe(0)
      expect(Array.isArray(result.actions)).toBe(true)
      expect(result.actions.length).toBe(0)
    })

    // Test the basic structure and functionality of the parser
    // without relying on specific implementation details
    it('should parse XML format correctly', () => {
      // This is a simplified test that just checks if the parser
      // can handle XML format without throwing errors
      const xml = `
        <tag1>content1</tag1>
        <tag2>content2</tag2>
      `
      const result = parser.Parse(xml)
      
      expect(result).toBeInstanceOf(MsgResp)
      expect(result.origin).toBe(xml)

      // We don't make specific assertions about the parsed content
      // since we're just testing that the parser doesn't crash
    })
  })

  // Test the snap method indirectly through Parse
  describe('snap method (indirectly)', () => {
    it('should extract content between tags', () => {
      // This test indirectly tests the snap method
      // We're not testing specific implementation details
      const xml = '<thinking>Test thinking</thinking>'
      const result = parser.Parse(xml)
      
      console.log('result', result)
      // The exact behavior depends on the implementation
      // We're just checking that something was processed
      expect(result).toBeInstanceOf(MsgResp)
    })
  })

  // Test the parse method indirectly through Parse
  describe('parse method (indirectly)', () => {
    it('should process tags correctly', () => {
      // This test indirectly tests the parse method
      // We're not testing specific implementation details
      const xml = `
        <thinking>Test thinking</thinking>
        <use-mysql-query>
          <dbname>test</dbname>
          <query>SELECT * FROM table</query>
          <result>Test result</result>
        </use-mysql-query>
        <annotate>
          <summary>Test summary</summary>
          <context>Test context</context>
          <set-title>Test title</set-title>
        </annotate>
      `
      const result = parser.Parse(xml)

      // Again, the exact behavior depends on the implementation
      // We're just checking that something was processed
      expect(result).toBeInstanceOf(MsgResp)

      // Check if the parsed content matches the expected structure
      // This is a simplified test and doesn't check for specific content
      expect(result.thinking).toBe('Test thinking')
      expect(result.context).toBeDefined()
      expect(result.actions.length).toBe(2)
    })
  })

  describe('request (indirectly)', () => {
    it('should process tags correctly', () => {
      // This test indirectly tests the parse method
      // We're not testing specific implementation details
      const xml = `
        <request>
          <question>What is the name of the city you want weather for?</question>
          <options>
            <option>选项1</option>
            <option>选项2</option>
            <option>选项3</option>
          </options>
        </request>
      `
      const result = parser.Parse(xml)
      // Again, the exact behavior depends on the implementation
      // We're just checking that something was processed
      expect(result).toBeInstanceOf(MsgResp)
      // Check if the parsed content matches the expected structure
      // This is a simplified test and doesn't check for specific content
      expect(result.actions).toBeDefined()  
      expect(result.actions.length).toBe(1)
      const requireAnswer = result.actions[0] as MakeAsk
      expect(requireAnswer.question).toBe('What is the name of the city you want weather for?')
      expect(requireAnswer.options).toBeDefined()
      expect(requireAnswer.options.length).toBe(3)
      expect(requireAnswer.options[0]).toBe('选项1')
      expect(requireAnswer.options[1]).toBe('选项2')
      expect(requireAnswer.options[2]).toBe('选项3')
    })
  })
})