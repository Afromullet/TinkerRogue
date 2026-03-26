# Bug Fixes

- There has to be a tradeoff between casting spells and moving squads/engaging in combat. Doing both will feel too mechanical. Maybe instead give units the ability to cast spells
instead of their regular action. Although this will be hard to work in - what determines what spells a unit can cast? Probably just the leader. This also means that
we need a better system which determines what units can cast what spells (Should be defined in a JSON file). 
- XP Awards seem to be odd. I.E, didivided by 10. Combat resolution grants 40 xp, but units only get 4 xp
- Start Random Encounter alwys starts the same encounter 
- Can cast spells and use artifacts on enemy turn



# Combat

- Allow multiple factions to be part of combat. 

# GUI Updates

- Squad Edit Mode requires me to select a unit from the panel on the right before Removing, Making it a leader, or viewing the unit. I also want to be able to access that by selecting the unit in the grid
- Unit Purchase Mode needs to show Unit Info

# JSON

- Add Error Checking for JSON. I.E, a "Support" unit has to have a "Heal" attack type, and so on. Determine valid state
- Review other JSON files and determine valid state and do load-time checking.

# Raid Package

- Either define archetypes.go in a json file, or use the encounter system. Consider creating a variation of the encounter system which weighs certain units more
    * I.E, "Ambush" - fast units have a weight, ranged battery prioritizes ranged units, etc. 

# Input

- Allow controsl to be remapped

# Combat


- Add Zones of Control


# Action Evaluator

# Other 

- combat life cycle manager changes only support one player at the moment, de to storing the playerEntityID. Consider changing that for multiplayer
- ExecuteThreatEvolutionEffect needs to changed base off threat type/data.  
- Add Debug command to reload JSON. Will be used for testing
- I added a json file for raid room archetypes - gamedata\raidarchetypes.go. I will have to add some more randomization to the archetypes. One possibility is through unit traits. Not sure exactly yet how to implement that..Might just go with a "preferred room" field for the unit json. 

# Cleanup

- Determine whether you can use DirtyCache as an interface for all of the other caches. Also determine whether we really need the caches

- Make sure entities are cleaned up upon destruction. Need to determine what entities have a "lifecycle" by determining what addcomponent is called on. 

- combatqueries.go still has functions which search the entire ecs space


- JSON file error handling

- Testing package has some functions which create initial player commanders and squads. move them to another package - probably bootstrap

- type EventType int in Overworld types is growing too large

- Lots of Empty Claude relate directories. See if there are any issues when removing them

- Re-enable victory conditions in tickmanager.go. They are disabled for testing

- Destroyed factions need to be removed from combat

- Ranged heavy enemies only flee. They don't attack. Not hit and run. Just annoying

# Review

- Either encounters or behavior seem to be too skewed towards ranged threats. Using the threat layer visualization as baseline. Investigate whether it's the squad makup, or the weighting of attacks. Find a way to measure how effective the encounter creation and threat maps are
- Determine if GUI state management is clean enough

## The following packages need review

- Guioverworld

- GUISquads
- GUIOverworld


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