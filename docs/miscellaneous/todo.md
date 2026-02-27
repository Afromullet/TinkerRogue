# Bug Fixes

- XP Awards seem to be odd. I.E, didivided by 10. Combat resolution grants 40 xp, but units only get 4 xp
- Can always move the move undo command in combat, even after attacking
- Start Random Encounter alwys starts the same encounter 
- Raids when "Retreating" are not handled correctly.

# Combat

- Allow multiple factions to be part of combat. 

# GUI Updates

- Squad Edit Mode requires me to select a unit from the panel on the right before Removing, Making it a leader, or viewing the unit. I also want to be able to access that by selecting the unit in the grid
- Unit Purchase Mode needs to show Unit Info



# Raid Package

- Either define archetypes.go in a json file, or use the encounter system. Consider creating a variation of the encounter system which weighs certain units more
    * I.E, "Ambush" - fast units have a weight, ranged battery prioritizes ranged units, etc. 

# Combat

- Add a "new type of attack" to combat, which is a heal. Use targeting cells to make things easier  
- Add Zones of Control


# Action Evaluator

# Other 

- combat life cycle manager changes only support one player at the moment, de to storing the playerEntityID. Consider changing that for multiplayer
- There has to be a tradeoff between casting spells and moving squads/engaging in combat. Doing both will feel too mechanical. 
- ExecuteThreatEvolutionEffect needs to changed base off threat type/data.  

# Cleanup

- Make sure entities are cleaned up upon destruction. Need to determine what entities have a "lifecycle" by determining what addcomponent is called on. 

- combatqueries.go still has functions which search the entire ecs space

- Cleaner GUI input handling

- Review all consts to see if they should be defined in a json file

- encounter service should not have mode coordinator. It overlaps too much with what the GUI is doing (pending. Determine if this is necessary)

- JSON file error handling

- Testing package has some functions which create initial player commanders and squads. move them to another package - probably bootstrap

- type EventType int in Overworld types is growing too large

- Lots of Empty Claude relate directories. See if there are any issues when removing them

- Re-enable victory conditions in tickmanager.go. They are disabled for testing

# Review

- Either encounters or behavior seem to be too skewed towards ranged threats. Using the threat layer visualization as baseline. Investigate whether it's the squad makup, or the weighting of attacks. Find a way to measure how effective the encounter creation and threat maps are
- Determine if GUI state management is clean enough

## The following packages need review

- Guioverworld

- GUISquads
- GUIOverworld

- Worldmap
- Guiraids
- Raids


# Redundancies?

- Consider removing  fmt.Errorf statements throughout the code, such as in combatactionsystem and use an error log instead. Check into logging to see if we can disable them if possible. 


# Research 

- Explore the go fix command


# Overworld (Defer)

- System for gaining overworld resources. Whether it's through battle or territory tbd
- Overworld node spawning is far too clustered together. All nodes spawn close to eachother. 
- Consider using an interface for intent in overworld